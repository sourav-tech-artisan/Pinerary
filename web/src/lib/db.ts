import Dexie, { type EntityTable } from "dexie";
import type {
  AppSetting,
  LocalJourney,
  LocalLocationSample,
  LocalPhoto,
  LocalPlace,
  LocalStop,
  OutboxRecord,
} from "@/lib/types";

export class PineraryDatabase extends Dexie {
  journeys!: EntityTable<LocalJourney, "localId">;
  stops!: EntityTable<LocalStop, "localId">;
  places!: EntityTable<LocalPlace, "localId">;
  samples!: EntityTable<LocalLocationSample, "sampleId">;
  photos!: EntityTable<LocalPhoto, "localId">;
  outbox!: EntityTable<OutboxRecord, "id">;
  settings!: EntityTable<AppSetting, "key">;

  constructor(name = "pinerary") {
    super(name);
    this.version(1).stores({
      journeys: "localId, serverId, ownerKey, status, startedAt, [ownerKey+status]",
      stops: "localId, serverId, ownerKey, journeyLocalId, [journeyLocalId+sequence], capturedAt",
      places: "localId, serverId, clientRequestId, ownerKey, createdAt",
      samples: "sampleId, ownerKey, journeyLocalId, [journeyLocalId+synced], capturedAt",
      photos: "localId, serverId, ownerKey, journeyLocalId, stopLocalId, syncState",
      outbox: "id, ownerKey, operation, entityLocalId, journeyLocalId, state, nextAttemptAt, createdAt, [ownerKey+state]",
      settings: "key",
    });
  }
}

export const db = new PineraryDatabase();
