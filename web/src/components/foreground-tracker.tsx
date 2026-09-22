"use client";

import { Crosshair, Pause, Satellite } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { distanceMeters } from "@/lib/geo";
import { recordLocation } from "@/lib/repository";
import type { LocalJourney } from "@/lib/types";

type TrackerState = "starting" | "recording" | "paused" | "denied" | "unavailable";

export function ForegroundTracker({ journey }: { journey: LocalJourney }) {
  const [state, setState] = useState<TrackerState>("starting");
  const [accuracy, setAccuracy] = useState<number>();
  const lastAccepted = useRef<{ latitude: number; longitude: number; at: number } | undefined>(undefined);

  useEffect(() => {
    if (journey.status !== "active") return;
    if (!("geolocation" in navigator)) {
      queueMicrotask(() => setState("unavailable"));
      return;
    }
    const visibility = () => setState(document.hidden ? "paused" : "recording");
    document.addEventListener("visibilitychange", visibility);
    const watchID = navigator.geolocation.watchPosition(
      (position) => {
        setAccuracy(position.coords.accuracy);
        setState(document.hidden ? "paused" : "recording");
        const previous = lastAccepted.current;
        const elapsed = previous ? position.timestamp - previous.at : Number.POSITIVE_INFINITY;
        const moved = previous
          ? distanceMeters(previous, { latitude: position.coords.latitude, longitude: position.coords.longitude })
          : Number.POSITIVE_INFINITY;
        if (elapsed < 15_000 || (moved < 12 && elapsed < 45_000)) return;
        lastAccepted.current = {
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          at: position.timestamp,
        };
        void recordLocation(journey.localId, position.coords, new Date(position.timestamp));
      },
      (error) => setState(error.code === error.PERMISSION_DENIED ? "denied" : "unavailable"),
      { enableHighAccuracy: true, maximumAge: 8_000, timeout: 20_000 },
    );
    return () => {
      navigator.geolocation.clearWatch(watchID);
      document.removeEventListener("visibilitychange", visibility);
    };
  }, [journey.localId, journey.status]);

  const copy = {
    starting: "Starting GPS…",
    recording: accuracy ? `GPS ±${Math.round(accuracy)} m` : "Recording route",
    paused: "Tracking may pause in background",
    denied: "Location permission denied",
    unavailable: "Location temporarily unavailable",
  }[state];
  const Icon = state === "paused" ? Pause : state === "recording" ? Satellite : Crosshair;

  return <div className={`tracker-status ${state}`} role="status"><Icon size={16} /> {copy}</div>;
}
