import type { components } from "@/lib/api/schema";

export type JourneyKind = components["schemas"]["JourneyKind"];
export type JourneyStatus = components["schemas"]["JourneyStatus"];
export type TransportMode = components["schemas"]["TransportMode"];
export type JourneyDTO = components["schemas"]["Journey"];
export type StopDTO = components["schemas"]["JourneyStop"];
export type PlaceDTO = components["schemas"]["Place"];
export type RouteSegment = components["schemas"]["RouteSegment"];
export type NearbyResult = components["schemas"]["NearbyResult"];
export type PhotoDTO = components["schemas"]["Photo"];
export type CreatedShare = components["schemas"]["CreatedShare"];
export type UserDTO = components["schemas"]["User"];

export type SyncState = "pending" | "synced" | "failed";

export interface LocalJourney {
  localId: string;
  ownerKey: string;
  serverId?: string;
  clientRequestId: string;
  kind: JourneyKind;
  label: string;
  status: JourneyStatus;
  startedAt: string;
  endedAt?: string;
  version: number;
  syncState: SyncState;
}

export interface LocalStop {
  localId: string;
  ownerKey: string;
  serverId?: string;
  placeServerId?: string;
  journeyLocalId: string;
  clientRequestId: string;
  sequence: number;
  capturedAt: string;
  name: string;
  note: string;
  latitude: number;
  longitude: number;
  syncState: SyncState;
}

export interface LocalPlace {
  localId: string;
  ownerKey: string;
  serverId?: string;
  clientRequestId: string;
  name: string;
  notes: string;
  latitude: number;
  longitude: number;
  createdAt: string;
  updatedAt: string;
  syncState: SyncState;
}

export interface LocalLocationSample {
  sampleId: string;
  ownerKey: string;
  journeyLocalId: string;
  capturedAt: string;
  latitude: number;
  longitude: number;
  accuracyM?: number;
  speedMPS?: number;
  headingDeg?: number;
  synced: 0 | 1;
}

export interface LocalPhoto {
  localId: string;
  ownerKey: string;
  serverId?: string;
  journeyLocalId: string;
  stopLocalId: string;
  contentType: "image/jpeg" | "image/png" | "image/webp";
  capturedAt: string;
  blob?: Blob;
  thumbnailURL?: string;
  syncState: SyncState;
}

export type OutboxOperation =
  | "create_journey"
  | "update_journey"
  | "end_journey"
  | "pin_stop"
  | "update_stop"
  | "save_place"
  | "upload_locations"
  | "upload_photo";

export type OutboxState = "pending" | "sending" | "failed";

export interface OutboxRecord {
  id: string;
  ownerKey: string;
  operation: OutboxOperation;
  entityLocalId: string;
  journeyLocalId?: string;
  state: OutboxState;
  attempts: number;
  createdAt: number;
  nextAttemptAt: number;
  lastError?: string;
  permanent?: boolean;
}

export interface AppSetting {
  key: string;
  value: unknown;
}

export interface Coordinates {
  latitude: number;
  longitude: number;
  accuracy?: number;
}
