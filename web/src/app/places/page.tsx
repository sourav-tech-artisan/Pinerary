"use client";

import { useLiveQuery } from "dexie-react-hooks";
import { ExternalLink, MapPin, Plus, Search } from "lucide-react";
import { useMemo, useState } from "react";
import { useApp } from "@/components/app-provider";
import { EmptyState, InlineMessage, PageHeading, SyncBadge } from "@/components/ui";
import { reverseGeocode } from "@/lib/api/client";
import { getOwnerKey } from "@/lib/config";
import { db } from "@/lib/db";
import { currentCoordinates } from "@/lib/geo";
import { saveStandalonePlace } from "@/lib/repository";
import type { Coordinates } from "@/lib/types";

export default function PlacesPage() {
  const ownerKey = getOwnerKey();
  const { syncNow } = useApp();
  const places = useLiveQuery(() => db.places.where("ownerKey").equals(ownerKey).toArray(), [ownerKey]);
  const [query, setQuery] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [coordinates, setCoordinates] = useState<Coordinates>();
  const [name, setName] = useState("");
  const [notes, setNotes] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const filtered = useMemo(
    () => (places ?? []).filter((place) => `${place.name} ${place.notes}`.toLowerCase().includes(query.toLowerCase())).sort((a, b) => b.createdAt.localeCompare(a.createdAt)),
    [places, query],
  );

  async function beginCapture() {
    setShowForm(true);
    setBusy(true);
    setError("");
    try {
      const location = await currentCoordinates();
      setCoordinates(location);
      const suggestion = navigator.onLine
        ? await reverseGeocode(location.latitude, location.longitude).catch(() => undefined)
        : undefined;
      setName(suggestion?.display_name || "Saved place");
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not capture your location.");
    } finally {
      setBusy(false);
    }
  }

  async function save(event: React.FormEvent) {
    event.preventDefault();
    if (!coordinates || !name.trim()) return;
    await saveStandalonePlace(name, notes, coordinates);
    setShowForm(false);
    setCoordinates(undefined);
    setName("");
    setNotes("");
    void syncNow();
  }

  return (
    <div className="page">
      <PageHeading
        eyebrow="Private place library"
        title="Places worth returning to"
        description="Save somewhere now or collect places before a future trip. Every pin remains available offline."
        action={<button className="button primary" onClick={beginCapture}><Plus size={18} /> Pin current place</button>}
      />

      <label className="search-box">
        <Search size={18} />
        <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search your saved places" />
      </label>

      {filtered.length ? (
        <section className="place-grid">
          {filtered.map((place) => (
            <article className="place-card" key={place.localId}>
              <div className="place-card-pin"><MapPin size={20} /></div>
              <div>
                <div className="place-card-heading"><h2>{place.name}</h2><SyncBadge state={place.syncState} /></div>
                {place.notes && <p>{place.notes}</p>}
                <small>{place.latitude.toFixed(5)}, {place.longitude.toFixed(5)}</small>
              </div>
              <a href={`https://www.openstreetmap.org/?mlat=${place.latitude}&mlon=${place.longitude}#map=16/${place.latitude}/${place.longitude}`} target="_blank" rel="noreferrer" aria-label={`Open ${place.name} on map`}><ExternalLink size={17} /></a>
            </article>
          ))}
        </section>
      ) : (
        <EmptyState title={query ? "No matching places" : "Build your personal map"} copy={query ? "Try another name or note." : "Pin a café, viewpoint, hotel, or any place you want to find again."} />
      )}

      {showForm && (
        <div className="modal-backdrop" role="presentation" onMouseDown={(event) => event.target === event.currentTarget && setShowForm(false)}>
          <section className="modal-card" role="dialog" aria-modal="true" aria-labelledby="place-title">
            <button className="modal-close" onClick={() => setShowForm(false)} aria-label="Close">×</button>
            <p className="eyebrow">Standalone pin</p>
            <h2 id="place-title">Save where you are</h2>
            {busy && <InlineMessage>Getting an accurate location…</InlineMessage>}
            {error && <InlineMessage tone="error">{error}</InlineMessage>}
            {coordinates && (
              <form onSubmit={save}>
                <p className="coordinate-preview"><MapPin size={16} /> {coordinates.latitude.toFixed(5)}, {coordinates.longitude.toFixed(5)} · ±{Math.round(coordinates.accuracy ?? 0)} m</p>
                <label className="field"><span>Name</span><input value={name} maxLength={200} onChange={(event) => setName(event.target.value)} /></label>
                <label className="field"><span>Notes</span><textarea value={notes} rows={3} onChange={(event) => setNotes(event.target.value)} placeholder="Why should you come back?" /></label>
                <button className="button primary wide"><MapPin size={18} /> Save place</button>
              </form>
            )}
          </section>
        </div>
      )}
    </div>
  );
}
