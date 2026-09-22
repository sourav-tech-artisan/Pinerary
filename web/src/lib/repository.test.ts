import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { db } from "@/lib/db";
import { finishJourney, pinJourneyStop, saveStandalonePlace, startJourney } from "@/lib/repository";

describe("offline-first repository", () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
  });

  afterEach(async () => {
    await db.delete();
  });

  it("atomically stores a new journey and its outbox mutation", async () => {
    const journey = await startJourney("trip", "Delhi ride");

    expect(await db.journeys.get(journey.localId)).toMatchObject({ label: "Delhi ride", status: "active" });
    expect(await db.outbox.toArray()).toEqual([
      expect.objectContaining({ operation: "create_journey", entityLocalId: journey.localId, state: "pending" }),
    ]);
    await expect(startJourney("outing", "Second active journey")).rejects.toThrow("Finish the active journey");
  });

  it("preserves stop sequence and queues completion after captured work", async () => {
    const journey = await startJourney("trip", "Goa");
    const first = await pinJourneyStop(journey.localId, "Beach", "Sunset", { latitude: 15.50, longitude: 73.77 });
    const second = await pinJourneyStop(journey.localId, "Fort", "Morning", { latitude: 15.49, longitude: 73.76 });
    await finishJourney(journey.localId);

    expect(first.sequence).toBe(1);
    expect(second.sequence).toBe(2);
    expect((await db.journeys.get(journey.localId))?.status).toBe("ended");
    expect((await db.outbox.orderBy("createdAt").toArray()).map((item) => item.operation)).toEqual([
      "create_journey",
      "pin_stop",
      "pin_stop",
      "end_journey",
    ]);
  });

  it("stores a standalone place immediately while offline", async () => {
    const place = await saveStandalonePlace("Hidden café", "Good coffee", { latitude: 28.6, longitude: 77.2 });
    expect(await db.places.get(place.localId)).toMatchObject({ name: "Hidden café", syncState: "pending" });
    expect(await db.outbox.where("operation").equals("save_place").count()).toBe(1);
  });
});
