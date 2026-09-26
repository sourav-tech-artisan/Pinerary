import type { PlaceSearchResult } from "@/lib/types";

export type PlaceSearchExecutor = (query: string, signal: AbortSignal) => Promise<PlaceSearchResult>;

export interface PlaceSearchCallbacks {
  onReset: () => void;
  onLoading: (query: string) => void;
  onSuccess: (query: string, result: PlaceSearchResult) => void;
  onError: (query: string, error: unknown) => void;
}

export function normalizePlaceQuery(value: string): string {
  return value.trim().replace(/\s+/gu, " ");
}

function wasAborted(error: unknown): boolean {
  return error instanceof DOMException
    ? error.name === "AbortError"
    : error instanceof Error && error.name === "AbortError";
}

/**
 * Owns autocomplete timing and request cancellation outside React so the
 * network behavior stays deterministic and independently testable.
 */
export class PlaceSearchCoordinator {
  private timer?: ReturnType<typeof setTimeout>;
  private controller?: AbortController;
  private generation = 0;

  constructor(
    private readonly execute: PlaceSearchExecutor,
    private readonly delayMilliseconds = 450,
  ) {}

  schedule(value: string, callbacks: PlaceSearchCallbacks): void {
    this.cancel();
    const query = normalizePlaceQuery(value);
    if (query.length < 3) {
      callbacks.onReset();
      return;
    }
    this.timer = setTimeout(() => {
      this.timer = undefined;
      void this.start(query, callbacks);
    }, this.delayMilliseconds);
  }

  runNow(value: string, callbacks: PlaceSearchCallbacks): Promise<void> {
    this.cancel();
    const query = normalizePlaceQuery(value);
    if (query.length < 3) {
      callbacks.onReset();
      return Promise.resolve();
    }
    return this.start(query, callbacks);
  }

  cancel(): void {
    if (this.timer !== undefined) clearTimeout(this.timer);
    this.timer = undefined;
    this.controller?.abort();
    this.controller = undefined;
    this.generation += 1;
  }

  private async start(query: string, callbacks: PlaceSearchCallbacks): Promise<void> {
    const generation = ++this.generation;
    const controller = new AbortController();
    this.controller = controller;
    callbacks.onLoading(query);
    try {
      const result = await this.execute(query, controller.signal);
      if (generation === this.generation) callbacks.onSuccess(query, result);
    } catch (error) {
      if (generation === this.generation && !wasAborted(error)) callbacks.onError(query, error);
    } finally {
      if (generation === this.generation) this.controller = undefined;
    }
  }
}
