import { APIError } from "@/lib/api/client";
import * as api from "@/lib/api/client";
import { getOwnerKey } from "@/lib/config";
import { db } from "@/lib/db";
import { newID } from "@/lib/id";
import type { JourneyDTO, LocalJourney, LocalPlace, LocalStop, OutboxRecord } from "@/lib/types";

let activeSync: Promise<SyncReport> | undefined;

export interface SyncReport {
  processed: number;
  failed: number;
  pending: number;
}

class MissingDependencyError extends Error {}

function journeyFromServer(remote: JourneyDTO, existing?: LocalJourney): LocalJourney {
  return {
    localId: existing?.localId ?? `server-${remote.id}`,
    ownerKey: getOwnerKey(),
    serverId: remote.id,
    clientRequestId: existing?.clientRequestId ?? newID(),
    kind: remote.kind,
    label: remote.label,
    status: remote.status,
    startedAt: remote.started_at,
    endedAt: remote.ended_at ?? undefined,
    version: remote.version,
    syncState: "synced",
  };
}

async function updateJourneyFromServer(localId: string, remote: JourneyDTO): Promise<void> {
  const existing = await db.journeys.get(localId);
  await db.journeys.put(journeyFromServer(remote, existing));
}

export async function pullRemoteData(): Promise<void> {
  if (typeof navigator !== "undefined" && !navigator.onLine) return;
  const ownerKey = getOwnerKey();
  const [journeys, places] = await Promise.all([api.listJourneys(), api.listPlaces()]);
  await db.transaction("rw", db.journeys, db.places, async () => {
    for (const remote of journeys) {
      const existing = await db.journeys.where("serverId").equals(remote.id).first();
      await db.journeys.put(journeyFromServer(remote, existing));
    }
    for (const remote of places) {
      const existing =
        (await db.places.where("serverId").equals(remote.id).first()) ??
        (await db.places.where("clientRequestId").equals(remote.client_request_id).first());
      const local: LocalPlace = {
        localId: existing?.localId ?? `server-${remote.id}`,
        ownerKey,
        serverId: remote.id,
        clientRequestId: remote.client_request_id,
        name: remote.name,
        notes: remote.notes,
        latitude: remote.latitude,
        longitude: remote.longitude,
        createdAt: remote.created_at,
        updatedAt: remote.updated_at,
        syncState: "synced",
      };
      await db.places.put(local);
    }
  });
}

export async function pullJourneyStops(journeyLocalId: string): Promise<void> {
  const journey = await db.journeys.get(journeyLocalId);
  if (!journey?.serverId || (typeof navigator !== "undefined" && !navigator.onLine)) return;
  const remoteStops = await api.listStops(journey.serverId);
  await db.transaction("rw", db.stops, db.places, async () => {
    for (const remote of remoteStops) {
      const existing =
        (await db.stops.where("serverId").equals(remote.id).first()) ??
        (await db.stops.where("journeyLocalId").equals(journeyLocalId).and(
          (stop) => stop.clientRequestId === remote.client_request_id,
        ).first());
      const local: LocalStop = {
        localId: existing?.localId ?? `server-${remote.id}`,
        ownerKey: journey.ownerKey,
        serverId: remote.id,
        placeServerId: remote.place_id,
        journeyLocalId,
        clientRequestId: remote.client_request_id,
        sequence: remote.sequence_number,
        capturedAt: remote.captured_at,
        name: remote.name,
        note: remote.note,
        latitude: remote.latitude,
        longitude: remote.longitude,
        syncState: "synced",
      };
      await db.stops.put(local);
      const place = await db.places.where("clientRequestId").equals(remote.client_request_id).first();
      if (place) await db.places.update(place.localId, { serverId: remote.place_id, syncState: "synced" });
    }
  });
}

async function checksum(blob: Blob): Promise<string> {
  const digest = await crypto.subtle.digest("SHA-256", await blob.arrayBuffer());
  return [...new Uint8Array(digest)].map((value) => value.toString(16).padStart(2, "0")).join("");
}

async function dispatch(record: OutboxRecord): Promise<boolean> {
  switch (record.operation) {
    case "create_journey": {
      const journey = await db.journeys.get(record.entityLocalId);
      if (!journey) return true;
      const remote = await api.createJourney({
        client_request_id: journey.clientRequestId,
        kind: journey.kind,
        label: journey.label,
      });
      await updateJourneyFromServer(journey.localId, remote);
      return true;
    }
    case "update_journey": {
      const journey = await db.journeys.get(record.entityLocalId);
      if (!journey) return true;
      if (!journey.serverId) throw new MissingDependencyError("Journey is waiting to be created.");
      const remote = await api.updateJourney(journey.serverId, journey.label, journey.version);
      await updateJourneyFromServer(journey.localId, remote);
      return true;
    }
    case "end_journey": {
      const journey = await db.journeys.get(record.entityLocalId);
      if (!journey) return true;
      if (!journey.serverId) throw new MissingDependencyError("Journey is waiting to be created.");
      const remote = await api.endJourney(journey.serverId, journey.version);
      await updateJourneyFromServer(journey.localId, remote);
      return true;
    }
    case "pin_stop": {
      const stop = await db.stops.get(record.entityLocalId);
      if (!stop) return true;
      const journey = await db.journeys.get(stop.journeyLocalId);
      if (!journey?.serverId) throw new MissingDependencyError("Journey is waiting to be created.");
      const remote = await api.pinStop(journey.serverId, {
        client_request_id: stop.clientRequestId,
        name: stop.name,
        note: stop.note,
        latitude: stop.latitude,
        longitude: stop.longitude,
        captured_at: stop.capturedAt,
      });
      await db.transaction("rw", db.stops, db.places, async () => {
        await db.stops.update(stop.localId, {
          serverId: remote.id,
          placeServerId: remote.place_id,
          sequence: remote.sequence_number,
          syncState: "synced",
        });
        const place = await db.places.where("clientRequestId").equals(stop.clientRequestId).first();
        if (place) await db.places.update(place.localId, { serverId: remote.place_id, syncState: "synced" });
      });
      return true;
    }
    case "update_stop": {
      const stop = await db.stops.get(record.entityLocalId);
      if (!stop) return true;
      const journey = await db.journeys.get(stop.journeyLocalId);
      if (!journey?.serverId || !stop.serverId) throw new MissingDependencyError("Stop is waiting to be created.");
      const remote = await api.updateStop(journey.serverId, stop.serverId, stop.name, stop.note);
      await db.stops.update(stop.localId, { name: remote.name, note: remote.note, syncState: "synced" });
      return true;
    }
    case "save_place": {
      const place = await db.places.get(record.entityLocalId);
      if (!place) return true;
      const remote = await api.savePlace({
        client_request_id: place.clientRequestId,
        name: place.name,
        notes: place.notes,
        latitude: place.latitude,
        longitude: place.longitude,
      });
      await db.places.update(place.localId, {
        serverId: remote.id,
        createdAt: remote.created_at,
        updatedAt: remote.updated_at,
        syncState: "synced",
      });
      return true;
    }
    case "upload_locations": {
      const journey = await db.journeys.get(record.entityLocalId);
      if (!journey) return true;
      if (!journey.serverId) throw new MissingDependencyError("Journey is waiting to be created.");
      const samples = await db.samples
        .where("[journeyLocalId+synced]")
        .equals([journey.localId, 0])
        .sortBy("capturedAt");
      const batch = samples.slice(0, 500);
      if (batch.length === 0) return true;
      await api.ingestLocations(
        journey.serverId,
        batch.map((sample) => ({
          sample_id: sample.sampleId,
          captured_at: sample.capturedAt,
          latitude: sample.latitude,
          longitude: sample.longitude,
          accuracy_m: sample.accuracyM,
          speed_mps: sample.speedMPS,
          heading_deg: sample.headingDeg,
        })),
      );
      await db.samples.bulkUpdate(batch.map((sample) => ({ key: sample.sampleId, changes: { synced: 1 as const } })));
      return samples.length <= 500;
    }
    case "upload_photo": {
      const photo = await db.photos.get(record.entityLocalId);
      if (!photo) return true;
      if (!photo.blob) return true;
      const journey = await db.journeys.get(photo.journeyLocalId);
      const stop = await db.stops.get(photo.stopLocalId);
      if (!journey?.serverId || !stop?.serverId) throw new MissingDependencyError("Photo is waiting for its stop.");
      const intent = await api.createPhotoIntent({
        journey_id: journey.serverId,
        stop_id: stop.serverId,
        client_request_id: photo.localId,
        content_type: photo.contentType,
        captured_at: photo.capturedAt,
      });
      if (photo.blob.size > intent.max_bytes) throw new Error("Photo exceeds the server upload limit.");
      await api.uploadPhoto(intent.upload_url, photo.blob, intent.content_type);
      await api.completePhoto(intent.photo_id, await checksum(photo.blob));
      await db.photos.update(photo.localId, { serverId: intent.photo_id, blob: undefined, syncState: "synced" });
      return true;
    }
  }
}

function permanentFailure(error: unknown): boolean {
  return error instanceof APIError && [400, 404, 409, 422].includes(error.status);
}

async function executeSync(): Promise<SyncReport> {
  if (typeof navigator !== "undefined" && !navigator.onLine) {
    return { processed: 0, failed: 0, pending: await db.outbox.count() };
  }
  const ownerKey = getOwnerKey();
  const records = await db.outbox
    .where("ownerKey")
    .equals(ownerKey)
    .and((record) => !record.permanent && record.nextAttemptAt <= Date.now())
    .sortBy("createdAt");
  let processed = 0;
  let failed = 0;
  for (const record of records) {
    await db.outbox.update(record.id, { state: "sending" });
    try {
      const complete = await dispatch(record);
      if (complete) {
        await db.outbox.delete(record.id);
      } else {
        await db.outbox.update(record.id, { state: "pending", nextAttemptAt: Date.now() });
      }
      processed += 1;
    } catch (error) {
      const dependency = error instanceof MissingDependencyError;
      const attempts = dependency ? record.attempts : record.attempts + 1;
      const delay = dependency ? 250 : Math.min(300_000, 1_000 * 2 ** Math.min(attempts, 8));
      await db.outbox.update(record.id, {
        state: "failed",
        attempts,
        lastError: error instanceof Error ? error.message : "Synchronization failed.",
        permanent: permanentFailure(error),
        nextAttemptAt: Date.now() + delay,
      });
      failed += 1;
      if (error instanceof APIError && [401, 403].includes(error.status)) break;
    }
  }
  return {
    processed,
    failed,
    pending: await db.outbox.where("ownerKey").equals(ownerKey).count(),
  };
}

export function synchronize(): Promise<SyncReport> {
  if (!activeSync) {
    activeSync = executeSync().finally(() => {
      activeSync = undefined;
    });
  }
  return activeSync;
}
