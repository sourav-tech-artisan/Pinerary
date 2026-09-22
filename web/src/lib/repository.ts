import { db } from "@/lib/db";
import { getOwnerKey } from "@/lib/config";
import { newID } from "@/lib/id";
import type {
  Coordinates,
  JourneyKind,
  LocalJourney,
  LocalLocationSample,
  LocalPhoto,
  LocalPlace,
  LocalStop,
  OutboxOperation,
} from "@/lib/types";

async function enqueue(
  operation: OutboxOperation,
  entityLocalId: string,
  journeyLocalId?: string,
): Promise<void> {
  const ownerKey = getOwnerKey();
  const existing = await db.outbox
    .where("entityLocalId")
    .equals(entityLocalId)
    .and((item) => item.ownerKey === ownerKey && item.operation === operation && !item.permanent)
    .first();
  if (existing) {
    await db.outbox.update(existing.id, { state: "pending", nextAttemptAt: Date.now() });
    return;
  }
  const dueAt = Date.now();
  const latest = await db.outbox.orderBy("createdAt").last();
  const createdAt = Math.max(dueAt, (latest?.createdAt ?? 0) + 1);
  await db.outbox.add({
    id: newID(),
    ownerKey,
    operation,
    entityLocalId,
    journeyLocalId,
    state: "pending",
    attempts: 0,
    createdAt,
    nextAttemptAt: dueAt,
  });
}

export async function startJourney(kind: JourneyKind, label: string): Promise<LocalJourney> {
  const ownerKey = getOwnerKey();
  const active = await db.journeys.where("[ownerKey+status]").equals([ownerKey, "active"]).first();
  if (active) throw new Error("Finish the active journey before starting another one.");

  const localId = newID();
  const journey: LocalJourney = {
    localId,
    ownerKey,
    clientRequestId: newID(),
    kind,
    label: label.trim(),
    status: "active",
    startedAt: new Date().toISOString(),
    version: 1,
    syncState: "pending",
  };
  await db.transaction("rw", db.journeys, db.outbox, async () => {
    await db.journeys.add(journey);
    await enqueue("create_journey", localId, localId);
  });
  return journey;
}

export async function renameJourney(localId: string, label: string): Promise<void> {
  await db.transaction("rw", db.journeys, db.outbox, async () => {
    await db.journeys.update(localId, { label: label.trim(), syncState: "pending" });
    await enqueue("update_journey", localId, localId);
  });
}

export async function finishJourney(localId: string): Promise<void> {
  const journey = await db.journeys.get(localId);
  if (!journey || journey.status !== "active") return;
  await db.transaction("rw", db.journeys, db.outbox, async () => {
    await db.journeys.update(localId, {
      status: "ended",
      endedAt: new Date().toISOString(),
      syncState: "pending",
    });
    await enqueue("end_journey", localId, localId);
  });
}

export async function pinJourneyStop(
  journeyLocalId: string,
  name: string,
  note: string,
  coordinates: Coordinates,
): Promise<LocalStop> {
  const journey = await db.journeys.get(journeyLocalId);
  if (!journey || journey.status !== "active") throw new Error("This journey is no longer active.");

  const previous = await db.stops.where("journeyLocalId").equals(journeyLocalId).sortBy("sequence");
  const clientRequestId = newID();
  const now = new Date().toISOString();
  const stop: LocalStop = {
    localId: newID(),
    ownerKey: journey.ownerKey,
    journeyLocalId,
    clientRequestId,
    sequence: (previous.at(-1)?.sequence ?? 0) + 1,
    capturedAt: now,
    name: name.trim(),
    note: note.trim(),
    latitude: coordinates.latitude,
    longitude: coordinates.longitude,
    syncState: "pending",
  };
  const place: LocalPlace = {
    localId: newID(),
    ownerKey: journey.ownerKey,
    clientRequestId,
    name: stop.name,
    notes: stop.note,
    latitude: stop.latitude,
    longitude: stop.longitude,
    createdAt: now,
    updatedAt: now,
    syncState: "pending",
  };
  await db.transaction("rw", db.stops, db.places, db.outbox, async () => {
    await db.stops.add(stop);
    await db.places.add(place);
    await enqueue("pin_stop", stop.localId, journeyLocalId);
  });
  return stop;
}

export async function editJourneyStop(localId: string, name: string, note: string): Promise<void> {
  const stop = await db.stops.get(localId);
  if (!stop) return;
  await db.transaction("rw", db.stops, db.outbox, async () => {
    await db.stops.update(localId, { name: name.trim(), note: note.trim(), syncState: "pending" });
    await enqueue("update_stop", localId, stop.journeyLocalId);
  });
}

export async function saveStandalonePlace(
  name: string,
  notes: string,
  coordinates: Coordinates,
): Promise<LocalPlace> {
  const now = new Date().toISOString();
  const place: LocalPlace = {
    localId: newID(),
    ownerKey: getOwnerKey(),
    clientRequestId: newID(),
    name: name.trim(),
    notes: notes.trim(),
    latitude: coordinates.latitude,
    longitude: coordinates.longitude,
    createdAt: now,
    updatedAt: now,
    syncState: "pending",
  };
  await db.transaction("rw", db.places, db.outbox, async () => {
    await db.places.add(place);
    await enqueue("save_place", place.localId);
  });
  return place;
}

export async function recordLocation(
  journeyLocalId: string,
  coordinates: GeolocationCoordinates,
  capturedAt = new Date(),
): Promise<LocalLocationSample> {
  const journey = await db.journeys.get(journeyLocalId);
  if (!journey || journey.status !== "active") throw new Error("No active journey to record.");
  const sample: LocalLocationSample = {
    sampleId: newID(),
    ownerKey: journey.ownerKey,
    journeyLocalId,
    capturedAt: capturedAt.toISOString(),
    latitude: coordinates.latitude,
    longitude: coordinates.longitude,
    accuracyM: Number.isFinite(coordinates.accuracy) ? coordinates.accuracy : undefined,
    speedMPS: coordinates.speed == null ? undefined : coordinates.speed,
    headingDeg: coordinates.heading == null ? undefined : coordinates.heading,
    synced: 0,
  };
  await db.transaction("rw", db.samples, db.outbox, async () => {
    await db.samples.add(sample);
    await enqueue("upload_locations", journeyLocalId, journeyLocalId);
  });
  return sample;
}

export async function queuePhoto(journeyLocalId: string, stopLocalId: string, file: File): Promise<LocalPhoto> {
  const allowed = ["image/jpeg", "image/png", "image/webp"] as const;
  if (!allowed.includes(file.type as (typeof allowed)[number])) {
    throw new Error("Choose a JPEG, PNG, or WebP image.");
  }
  if (file.size > 15 * 1024 * 1024) throw new Error("Photo must be smaller than 15 MB.");
  const journey = await db.journeys.get(journeyLocalId);
  if (!journey) throw new Error("Journey not found.");
  const photo: LocalPhoto = {
    localId: newID(),
    ownerKey: journey.ownerKey,
    journeyLocalId,
    stopLocalId,
    contentType: file.type as LocalPhoto["contentType"],
    capturedAt: new Date(file.lastModified || Date.now()).toISOString(),
    blob: file,
    syncState: "pending",
  };
  await db.transaction("rw", db.photos, db.outbox, async () => {
    await db.photos.add(photo);
    await enqueue("upload_photo", photo.localId, journeyLocalId);
  });
  return photo;
}
