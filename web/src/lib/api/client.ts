import createClient from "openapi-fetch";
import type { paths, components } from "@/lib/api/schema";
import { getAPIBaseURL, getAuthToken } from "@/lib/config";
import type { TransportMode } from "@/lib/types";

type ErrorEnvelope = components["schemas"]["ErrorEnvelope"];
type CreateJourneyRequest = components["schemas"]["CreateJourneyRequest"];
type PinStopRequest = components["schemas"]["PinStopRequest"];
type SavePlaceRequest = components["schemas"]["SavePlaceRequest"];
type LocationPoint = components["schemas"]["LocationPoint"];
type ShareRequest = components["schemas"]["CreateShareRequest"];
type PhotoIntentRequest = components["schemas"]["CreatePhotoUploadIntentRequest"];

export class APIError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code = "request_failed",
    readonly requestID?: string,
  ) {
    super(message);
    this.name = "APIError";
  }
}

function api() {
  return createClient<paths>({
    baseUrl: getAPIBaseURL(),
    headers: {
      Authorization: `Bearer ${getAuthToken()}`,
    },
  });
}

function requireData<T>(data: T | undefined, error: unknown, response: Response): T {
  if (data !== undefined) return data;
  const envelope = error as ErrorEnvelope | undefined;
  throw new APIError(
    envelope?.error?.message || `Request failed with status ${response.status}`,
    response.status,
    envelope?.error?.code,
    envelope?.error?.request_id,
  );
}

export async function getProfile() {
  const { data, error, response } = await api().GET("/me");
  return requireData(data, error, response);
}

export async function updateProfile(displayName: string, defaultTransportMode: TransportMode) {
  const { data, error, response } = await api().PATCH("/me", {
    body: { display_name: displayName, default_transport_mode: defaultTransportMode },
  });
  return requireData(data, error, response);
}

export async function registerDevice(installationID: string, pushSubscription: Record<string, unknown>) {
  const { data, error, response } = await api().POST("/devices", {
    body: { installation_id: installationID, platform: "web", push_subscription: pushSubscription },
  });
  return requireData(data, error, response);
}

export async function deleteDevice(deviceID: string) {
  const { error, response } = await api().DELETE("/devices/{deviceId}", {
    params: { path: { deviceId: deviceID } },
  });
  if (!response.ok) requireData(undefined, error, response);
}

export async function listJourneys() {
  const { data, error, response } = await api().GET("/journeys", {
    params: { query: { page_size: 100 } },
  });
  return requireData(data, error, response).items;
}

export async function createJourney(body: CreateJourneyRequest) {
  const { data, error, response } = await api().POST("/journeys", { body });
  return requireData(data, error, response);
}

export async function updateJourney(journeyID: string, label: string, version: number) {
  const { data, error, response } = await api().PATCH("/journeys/{journeyId}", {
    params: { path: { journeyId: journeyID } },
    body: { label, version },
  });
  return requireData(data, error, response);
}

export async function endJourney(journeyID: string, version: number) {
  const { data, error, response } = await api().POST("/journeys/{journeyId}/end", {
    params: { path: { journeyId: journeyID } },
    body: { version },
  });
  return requireData(data, error, response);
}

export async function listStops(journeyID: string) {
  const { data, error, response } = await api().GET("/journeys/{journeyId}/stops", {
    params: { path: { journeyId: journeyID } },
  });
  return requireData(data, error, response).items;
}

export async function pinStop(journeyID: string, body: PinStopRequest) {
  const { data, error, response } = await api().POST("/journeys/{journeyId}/stops", {
    params: { path: { journeyId: journeyID } },
    body,
  });
  return requireData(data, error, response);
}

export async function updateStop(journeyID: string, stopID: string, name: string, note: string) {
  const { data, error, response } = await api().PATCH("/journeys/{journeyId}/stops/{stopId}", {
    params: { path: { journeyId: journeyID, stopId: stopID } },
    body: { name, note },
  });
  return requireData(data, error, response);
}

export async function ingestLocations(journeyID: string, points: LocationPoint[]) {
  const { data, error, response } = await api().POST("/journeys/{journeyId}/locations/batch", {
    params: { path: { journeyId: journeyID } },
    body: { points },
  });
  return requireData(data, error, response);
}

export async function getRoute(journeyID: string) {
  const { data, error, response } = await api().GET("/journeys/{journeyId}/route", {
    params: { path: { journeyId: journeyID } },
  });
  return requireData(data, error, response).segments;
}

export async function listPlaces() {
  const { data, error, response } = await api().GET("/places", {
    params: { query: { limit: 500 } },
  });
  return requireData(data, error, response).items;
}

export async function savePlace(body: SavePlaceRequest) {
  const { data, error, response } = await api().POST("/places", { body });
  return requireData(data, error, response);
}

export async function reverseGeocode(latitude: number, longitude: number) {
  const { data, error, response } = await api().GET("/places/reverse-geocode", {
    params: { query: { latitude, longitude } },
  });
  return requireData(data, error, response);
}

export async function nearbyPlaces(latitude: number, longitude: number, mode: TransportMode) {
  const { data, error, response } = await api().GET("/places/nearby", {
    params: { query: { latitude, longitude, mode, limit: 10 } },
  });
  return requireData(data, error, response);
}

export async function createPhotoIntent(body: PhotoIntentRequest) {
  const { data, error, response } = await api().POST("/photos/upload-intents", { body });
  return requireData(data, error, response);
}

export async function uploadPhoto(uploadURL: string, blob: Blob, contentType: string) {
  const response = await fetch(uploadURL, {
    method: "PUT",
    headers: { "Content-Type": contentType },
    body: blob,
  });
  if (!response.ok) throw new APIError("Photo upload failed.", response.status, "photo_upload_failed");
}

export async function completePhoto(photoID: string, checksum: string) {
  const { data, error, response } = await api().POST("/photos/{photoId}/complete", {
    params: { path: { photoId: photoID } },
    body: { checksum_sha256: checksum },
  });
  return requireData(data, error, response);
}

export async function listStopPhotos(journeyID: string, stopID: string) {
  const { data, error, response } = await api().GET("/journeys/{journeyId}/stops/{stopId}/photos", {
    params: { path: { journeyId: journeyID, stopId: stopID } },
  });
  return requireData(data, error, response).items;
}

export async function createShare(journeyID: string, body: ShareRequest) {
  const { data, error, response } = await api().POST("/journeys/{journeyId}/shares", {
    params: { path: { journeyId: journeyID } },
    body,
  });
  return requireData(data, error, response);
}
