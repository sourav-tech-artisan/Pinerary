"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { pullRemoteData, synchronize, type SyncReport } from "@/lib/sync";

type ConnectionState = "online" | "offline";
type SyncState = "idle" | "syncing" | "error";

interface AppContextValue {
  connection: ConnectionState;
  syncState: SyncState;
  lastSync?: Date;
  lastReport?: SyncReport;
  syncNow: () => Promise<void>;
}

const AppContext = createContext<AppContextValue | undefined>(undefined);

export function AppProvider({ children }: { children: React.ReactNode }) {
  const [connection, setConnection] = useState<ConnectionState>(() =>
    typeof navigator === "undefined" || navigator.onLine ? "online" : "offline",
  );
  const [syncState, setSyncState] = useState<SyncState>("idle");
  const [lastSync, setLastSync] = useState<Date>();
  const [lastReport, setLastReport] = useState<SyncReport>();

  const syncNow = useCallback(async () => {
    if (!navigator.onLine) {
      setConnection("offline");
      return;
    }
    setSyncState("syncing");
    try {
      const report = await synchronize();
      await pullRemoteData();
      setLastReport(report);
      setLastSync(new Date());
      setSyncState(report.failed > 0 ? "error" : "idle");
    } catch {
      setSyncState("error");
    }
  }, []);

  useEffect(() => {
    if ("serviceWorker" in navigator) {
      navigator.serviceWorker.register("/sw.js", { scope: "/", updateViaCache: "none" }).catch(() => undefined);
    }
    const online = () => {
      setConnection("online");
      void syncNow();
    };
    const offline = () => setConnection("offline");
    window.addEventListener("online", online);
    window.addEventListener("offline", offline);
    queueMicrotask(() => void syncNow());
    const interval = window.setInterval(() => void syncNow(), 30_000);
    return () => {
      window.removeEventListener("online", online);
      window.removeEventListener("offline", offline);
      window.clearInterval(interval);
    };
  }, [syncNow]);

  const value = useMemo(
    () => ({ connection, syncState, lastSync, lastReport, syncNow }),
    [connection, syncState, lastSync, lastReport, syncNow],
  );
  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

export function useApp() {
  const value = useContext(AppContext);
  if (!value) throw new Error("useApp must be used inside AppProvider");
  return value;
}
