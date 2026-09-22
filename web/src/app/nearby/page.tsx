"use client";

import { useLiveQuery } from "dexie-react-hooks";
import { Bike, Car, Footprints, LocateFixed, MapPin } from "lucide-react";
import { useEffect, useState } from "react";
import { EmptyState, InlineMessage, PageHeading } from "@/components/ui";
import { getProfile, nearbyPlaces } from "@/lib/api/client";
import { getOwnerKey, getPreferredTransportMode, savePreferredTransportMode } from "@/lib/config";
import { db } from "@/lib/db";
import { currentCoordinates, distanceMeters, formatDistance, formatDuration } from "@/lib/geo";
import type { NearbyResult, TransportMode } from "@/lib/types";

const modes: { value: TransportMode; label: string; icon: typeof Bike }[] = [
  { value: "motorcycle", label: "Motorcycle", icon: Bike },
  { value: "car", label: "Car", icon: Car },
  { value: "walking", label: "Walking", icon: Footprints },
];

export default function NearbyPage() {
  const ownerKey = getOwnerKey();
  const localPlaces = useLiveQuery(() => db.places.where("ownerKey").equals(ownerKey).toArray(), [ownerKey]) ?? [];
  const [mode, setMode] = useState<TransportMode>("motorcycle");
  const [result, setResult] = useState<NearbyResult>();
  const [offlineResult, setOfflineResult] = useState<(typeof localPlaces[number] & { distance: number })[]>([]);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [approximate, setApproximate] = useState(false);

  useEffect(() => {
    queueMicrotask(() => setMode(getPreferredTransportMode()));
    if (!navigator.onLine) return;
    getProfile().then((profile) => {
      setMode(profile.default_transport_mode);
      savePreferredTransportMode(profile.default_transport_mode);
    }).catch(() => undefined);
  }, []);

  async function search(selectedMode = mode) {
    setBusy(true);
    setMessage("");
    setResult(undefined);
    setOfflineResult([]);
    try {
      const origin = await currentCoordinates();
      if (!navigator.onLine) {
        const sorted = localPlaces
          .map((place) => ({ ...place, distance: distanceMeters(origin, place) }))
          .sort((left, right) => left.distance - right.distance)
          .slice(0, 10);
        setOfflineResult(sorted);
        setApproximate(true);
        return;
      }
      setResult(await nearbyPlaces(origin.latitude, origin.longitude, selectedMode));
      setApproximate(false);
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : "Could not find nearby places.");
    } finally {
      setBusy(false);
    }
  }

  function changeMode(next: TransportMode) {
    setMode(next);
    savePreferredTransportMode(next);
    if (result) void search(next);
  }

  return (
    <div className="page">
      <PageHeading
        eyebrow="Road-aware discovery"
        title="What have you saved nearby?"
        description="Results are ranked by road distance—not a straight line—with motorcycle as your default."
        action={<button className="button primary" onClick={() => search()} disabled={busy}><LocateFixed size={18} /> {busy ? "Locating…" : "Find nearest"}</button>}
      />

      <div className="segmented-control transport-tabs" aria-label="Travel mode">
        {modes.map(({ value, label, icon: Icon }) => (
          <button key={value} className={mode === value ? "active" : ""} onClick={() => changeMode(value)}><Icon size={18} /> {label}</button>
        ))}
      </div>

      {message && <InlineMessage tone="error">{message}</InlineMessage>}
      {approximate && <InlineMessage>Offline results use approximate straight-line distance. Reconnect for road distance and travel time.</InlineMessage>}

      {result?.items.length ? (
        <div className="nearby-list">
          {result.items.map((place, index) => {
            const selected = place.routes[mode];
            return (
              <article className="nearby-card" key={place.place_id}>
                <span className="nearby-rank">{index + 1}</span>
                <div className="nearby-main"><h2>{place.name}</h2><p><MapPin size={15} /> {selected ? formatDistance(selected.distance_m) : "Road unavailable"} by {mode}</p></div>
                <div className="mode-estimates">
                  {modes.map(({ value, icon: Icon }) => place.routes[value] && (
                    <span className={value === mode ? "selected" : ""} key={value}><Icon size={15} /> {formatDuration(place.routes[value]!.duration_s)}</span>
                  ))}
                </div>
              </article>
            );
          })}
        </div>
      ) : offlineResult.length ? (
        <div className="nearby-list">
          {offlineResult.map((place, index) => (
            <article className="nearby-card" key={place.localId}>
              <span className="nearby-rank">{index + 1}</span><div className="nearby-main"><h2>{place.name}</h2><p><MapPin size={15} /> ≈ {formatDistance(place.distance)} straight-line</p></div>
            </article>
          ))}
        </div>
      ) : (
        <EmptyState title="Ready to search" copy="Tap Find nearest to rank up to ten saved places from your current location." />
      )}
    </div>
  );
}
