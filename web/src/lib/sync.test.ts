import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const apiMocks = vi.hoisted(() => ({ createJourney: vi.fn(), pinStop: vi.fn() }));

vi.mock("@/lib/api/client", () => ({
  APIError: class APIError extends Error {},
  createJourney: apiMocks.createJourney,
  pinStop: apiMocks.pinStop,
  listJourneys: vi.fn().mockResolvedValue([]),
  listPlaces: vi.fn().mockResolvedValue([]),
  updateJourney: vi.fn(),
  endJourney: vi.fn(),
  updateStop: vi.fn(),
  savePlace: vi.fn(),
  ingestLocations: vi.fn(),
  createPhotoIntent: vi.fn(),
  uploadPhoto: vi.fn(),
  completePhoto: vi.fn(),
  listStops: vi.fn().mockResolvedValue([]),
}));

import { db } from "@/lib/db";
import { pinJourneyStop, startJourney } from "@/lib/repository";
import { synchronize } from "@/lib/sync";

describe("outbox synchronization", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    await db.delete();
    await db.open();
  });

  afterEach(async () => {
    await db.delete();
  });

  it("resolves server IDs before sending dependent stop mutations", async () => {
    const journey = await startJourney("trip", "Dependency ordering");
    const stop = await pinJourneyStop(journey.localId, "First stop", "", { latitude: 28.61, longitude: 77.23 });
    apiMocks.createJourney.mockResolvedValue({
      id: "11111111-1111-4111-8111-111111111111",
      kind: "trip",
      label: "Dependency ordering",
      status: "active",
      started_at: journey.startedAt,
      ended_at: null,
      version: 1,
    });
    apiMocks.pinStop.mockResolvedValue({
      id: "22222222-2222-4222-8222-222222222222",
      journey_id: "11111111-1111-4111-8111-111111111111",
      place_id: "33333333-3333-4333-8333-333333333333",
      client_request_id: stop.clientRequestId,
      sequence_number: 1,
      captured_at: stop.capturedAt,
      name: stop.name,
      note: "",
      latitude: stop.latitude,
      longitude: stop.longitude,
    });

    const report = await synchronize();

    expect(report).toMatchObject({ processed: 2, failed: 0, pending: 0 });
    expect(apiMocks.createJourney).toHaveBeenCalledOnce();
    expect(apiMocks.pinStop).toHaveBeenCalledWith("11111111-1111-4111-8111-111111111111", expect.any(Object));
    expect((await db.stops.get(stop.localId))?.serverId).toBe("22222222-2222-4222-8222-222222222222");
  });
});
