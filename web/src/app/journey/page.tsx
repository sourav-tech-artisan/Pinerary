"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useLiveQuery } from "dexie-react-hooks";
import { ArrowLeft, CheckCircle2, Clock3, Flag, MapPin, Navigation, Share2 } from "lucide-react";
import { Suspense, useEffect, useMemo, useState } from "react";
import { useApp } from "@/components/app-provider";
import { JourneyMap } from "@/components/journey-map";
import { ShareComposer } from "@/components/share-composer";
import { TimelineStop } from "@/components/timeline-stop";
import { EmptyState, InlineMessage, LoadingState, SyncBadge } from "@/components/ui";
import { getRoute, reverseGeocode } from "@/lib/api/client";
import { db } from "@/lib/db";
import { currentCoordinates } from "@/lib/geo";
import { finishJourney, pinJourneyStop } from "@/lib/repository";
import { pullJourneyStops } from "@/lib/sync";
import type { Coordinates, RouteSegment } from "@/lib/types";

function JourneyScreen() {
  const search = useSearchParams();
  const localId = search.get("id") ?? "";
  const { connection, syncNow } = useApp();
  const journey = useLiveQuery(() => db.journeys.get(localId), [localId]);
  const stops = useLiveQuery(
    () => db.stops.where("journeyLocalId").equals(localId).sortBy("sequence"),
    [localId],
  ) ?? [];
  const samples = useLiveQuery(
    () => db.samples.where("journeyLocalId").equals(localId).sortBy("capturedAt"),
    [localId],
  ) ?? [];
  const photos = useLiveQuery(() => db.photos.where("journeyLocalId").equals(localId).toArray(), [localId]) ?? [];
  const [route, setRoute] = useState<RouteSegment[]>([]);
  const [pinDraft, setPinDraft] = useState<{ coordinates: Coordinates; name: string; note: string }>();
  const [locating, setLocating] = useState(false);
  const [message, setMessage] = useState("");
  const [showShare, setShowShare] = useState(false);

  useEffect(() => {
    if (!journey?.serverId || connection === "offline") return;
    void pullJourneyStops(journey.localId);
    const refreshRoute = () => getRoute(journey.serverId!).then(setRoute).catch(() => undefined);
    refreshRoute();
    if (journey.status !== "active") return;
    const interval = window.setInterval(refreshRoute, 20_000);
    return () => window.clearInterval(interval);
  }, [journey?.localId, journey?.serverId, journey?.status, connection]);

  const distance = useMemo(() => route.reduce((total, segment) => total + segment.distance_m, 0), [route]);

  if (!localId) return <EmptyState title="Journey not selected" copy="Return to your trips and choose a journey." action={<Link className="button primary" href="/">Back to trips</Link>} />;
  if (!journey) return <LoadingState label="Opening journey…" />;

  async function capturePin() {
    setMessage("");
    setLocating(true);
    try {
      const coordinates = await currentCoordinates();
      let name = "Pinned place";
      if (navigator.onLine) {
        name = (await reverseGeocode(coordinates.latitude, coordinates.longitude).catch(() => undefined))?.display_name || name;
      }
      setPinDraft({ coordinates, name, note: "" });
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : "Could not capture your location.");
    } finally {
      setLocating(false);
    }
  }

  async function savePin(event: React.FormEvent) {
    event.preventDefault();
    if (!pinDraft?.name.trim()) return;
    await pinJourneyStop(journey!.localId, pinDraft.name, pinDraft.note, pinDraft.coordinates);
    setPinDraft(undefined);
    void syncNow();
  }

  async function end() {
    if (!window.confirm("Finish this journey? Its timeline order and captured locations will remain fixed.")) return;
    await finishJourney(journey!.localId);
    await syncNow();
  }

  return (
    <div className="page journey-page">
      <div className="journey-topbar">
        <Link href="/" className="back-link"><ArrowLeft size={18} /> Trips</Link>
        <SyncBadge state={journey.syncState} />
      </div>

      <header className="journey-heading">
        <div>
          <p className="eyebrow">{journey.status === "active" ? `Active ${journey.kind}` : `${journey.kind} · ${journey.status}`}</p>
          <h1>{journey.label}</h1>
          <p>{new Intl.DateTimeFormat("en-IN", { dateStyle: "long", timeStyle: "short" }).format(new Date(journey.startedAt))}</p>
        </div>
        <div className="journey-actions">
          {journey.status === "active" ? (
            <>
              <button className="button primary" onClick={capturePin} disabled={locating}><MapPin size={18} /> {locating ? "Finding you…" : "Pin this place"}</button>
              <button className="button danger-quiet" onClick={end}><Flag size={18} /> Finish</button>
            </>
          ) : (
            <button className="button primary" onClick={() => setShowShare(true)} disabled={!journey.serverId || stops.some((stop) => !stop.serverId)}><Share2 size={18} /> Share itinerary</button>
          )}
        </div>
      </header>

      {message && <InlineMessage tone="error">{message}</InlineMessage>}

      <section className="map-panel">
        <JourneyMap samples={samples} stops={stops} route={route} />
        <div className="map-stats">
          <div><Navigation size={17} /><span><strong>{distance ? `${(distance / 1000).toFixed(1)} km` : "—"}</strong> processed route</span></div>
          <div><MapPin size={17} /><span><strong>{stops.length}</strong> stops</span></div>
          <div><Clock3 size={17} /><span><strong>{samples.length}</strong> GPS samples</span></div>
        </div>
      </section>

      <section className="timeline-section">
        <div className="section-heading">
          <div><p className="eyebrow">Captured in order</p><h2>Your timeline</h2></div>
          <span className="timeline-rule"><CheckCircle2 size={16} /> Order stays historical</span>
        </div>
        {stops.length ? (
          <div className="timeline">{stops.map((stop) => <TimelineStop key={stop.localId} stop={stop} journeyServerID={journey.serverId} />)}</div>
        ) : (
          <EmptyState
            title="No stops pinned yet"
            copy={journey.status === "active" ? "When somewhere feels worth remembering, pin it with one tap." : "This journey ended without any pinned stops."}
            action={journey.status === "active" ? <button className="button secondary" onClick={capturePin}><MapPin size={17} /> Pin your location</button> : undefined}
          />
        )}
      </section>

      {pinDraft && (
        <div className="modal-backdrop" role="presentation" onMouseDown={(event) => event.target === event.currentTarget && setPinDraft(undefined)}>
          <section className="modal-card" role="dialog" aria-modal="true" aria-labelledby="pin-title">
            <button className="modal-close" onClick={() => setPinDraft(undefined)} aria-label="Close">×</button>
            <p className="eyebrow">Captured now</p>
            <h2 id="pin-title">Save this stop</h2>
            <p className="coordinate-preview"><MapPin size={16} /> {pinDraft.coordinates.latitude.toFixed(5)}, {pinDraft.coordinates.longitude.toFixed(5)} · ±{Math.round(pinDraft.coordinates.accuracy ?? 0)} m</p>
            <form onSubmit={savePin}>
              <label className="field"><span>Name</span><input value={pinDraft.name} maxLength={200} onChange={(event) => setPinDraft({ ...pinDraft, name: event.target.value })} /></label>
              <label className="field"><span>Note</span><textarea value={pinDraft.note} rows={3} placeholder="What made this place memorable?" onChange={(event) => setPinDraft({ ...pinDraft, note: event.target.value })} /></label>
              <button className="button primary wide"><MapPin size={18} /> Add to timeline</button>
            </form>
          </section>
        </div>
      )}

      {showShare && journey.serverId && (
        <ShareComposer journeyID={journey.serverId} defaultTitle={journey.label} stops={stops} photos={photos} onClose={() => setShowShare(false)} />
      )}
    </div>
  );
}

export default function JourneyPage() {
  return <Suspense fallback={<LoadingState label="Opening journey…" />}><JourneyScreen /></Suspense>;
}
