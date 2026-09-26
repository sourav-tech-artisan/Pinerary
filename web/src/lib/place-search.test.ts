import { afterEach, describe, expect, it, vi } from "vitest";
import { normalizePlaceQuery, PlaceSearchCoordinator, type PlaceSearchCallbacks } from "@/lib/place-search";
import type { PlaceSearchResult } from "@/lib/types";

const emptyResult: PlaceSearchResult = {
  items: [],
  attribution: {
    provider: "Powered by Geoapify",
    provider_url: "https://www.geoapify.com/",
    data: "© OpenStreetMap contributors",
    data_url: "https://www.openstreetmap.org/copyright",
  },
};

function callbacks() {
  return {
    onReset: vi.fn(),
    onLoading: vi.fn(),
    onSuccess: vi.fn(),
    onError: vi.fn(),
  } satisfies PlaceSearchCallbacks;
}

describe("PlaceSearchCoordinator", () => {
  afterEach(() => vi.useRealTimers());

  it("normalizes whitespace and waits for a meaningful query", async () => {
    vi.useFakeTimers();
    const execute = vi.fn().mockResolvedValue(emptyResult);
    const coordinator = new PlaceSearchCoordinator(execute);
    const events = callbacks();

    expect(normalizePlaceQuery("  Bhappe   Da  ")).toBe("Bhappe Da");
    coordinator.schedule("ab", events);
    expect(events.onReset).toHaveBeenCalledOnce();

    coordinator.schedule("Bhappe Da", events);
    await vi.advanceTimersByTimeAsync(449);
    expect(execute).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(1);
    expect(execute).toHaveBeenCalledWith("Bhappe Da", expect.any(AbortSignal));
  });

  it("runs immediately on submit and cancels the pending debounce", async () => {
    vi.useFakeTimers();
    const execute = vi.fn().mockResolvedValue(emptyResult);
    const coordinator = new PlaceSearchCoordinator(execute);
    const events = callbacks();

    coordinator.schedule("Candolim beach", events);
    await coordinator.runNow("Candolim beach", events);
    await vi.runAllTimersAsync();

    expect(execute).toHaveBeenCalledOnce();
    expect(events.onSuccess).toHaveBeenCalledWith("Candolim beach", emptyResult);
  });

  it("ignores a superseded response even when its executor does not honor abort", async () => {
    const resolvers: Array<(result: PlaceSearchResult) => void> = [];
    const execute = vi.fn().mockImplementation(() => new Promise<PlaceSearchResult>((resolve) => resolvers.push(resolve)));
    const coordinator = new PlaceSearchCoordinator(execute);
    const events = callbacks();

    const first = coordinator.runNow("Delhi hotel", events);
    const second = coordinator.runNow("Goa hotel", events);
    resolvers[0](emptyResult);
    resolvers[1](emptyResult);
    await Promise.all([first, second]);

    expect(events.onSuccess).toHaveBeenCalledOnce();
    expect(events.onSuccess).toHaveBeenCalledWith("Goa hotel", emptyResult);
  });
});
