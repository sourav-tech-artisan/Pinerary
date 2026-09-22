import "fake-indexeddb/auto";

Object.defineProperty(globalThis, "navigator", {
  configurable: true,
  value: { onLine: true },
});
