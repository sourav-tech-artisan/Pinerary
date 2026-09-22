"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useLiveQuery } from "dexie-react-hooks";
import { ArrowRight, Bike, CalendarDays, MapPin, Plus, Route, Sparkles } from "lucide-react";
import { useState } from "react";
import { useApp } from "@/components/app-provider";
import { EmptyState, InlineMessage, LoadingState, PageHeading, SyncBadge } from "@/components/ui";
import { getOwnerKey } from "@/lib/config";
import { db } from "@/lib/db";
import { startJourney } from "@/lib/repository";
import type { JourneyKind, LocalJourney } from "@/lib/types";

function formatDate(value: string) {
  return new Intl.DateTimeFormat("en-IN", { day: "numeric", month: "short", year: "numeric" }).format(new Date(value));
}

function JourneyCard({ journey }: { journey: LocalJourney }) {
  return (
    <Link className={`journey-card ${journey.status === "active" ? "active" : ""}`} href={`/journey/?id=${journey.localId}`}>
      <div className="journey-card-icon">{journey.kind === "trip" ? <Route /> : <Bike />}</div>
      <div className="journey-card-copy">
        <div className="card-kicker">
          <span>{journey.kind}</span>
          <SyncBadge state={journey.syncState} />
        </div>
        <h3>{journey.label}</h3>
        <p><CalendarDays size={15} /> {formatDate(journey.startedAt)}</p>
      </div>
      <ArrowRight className="journey-arrow" size={20} />
    </Link>
  );
}

export default function HomePage() {
  const router = useRouter();
  const { syncNow } = useApp();
  const ownerKey = getOwnerKey();
  const journeys = useLiveQuery(async () => {
    const records = await db.journeys.where("ownerKey").equals(ownerKey).toArray();
    return records.sort((left, right) => right.startedAt.localeCompare(left.startedAt));
  }, [ownerKey]);
  const placesCount = useLiveQuery(() => db.places.where("ownerKey").equals(ownerKey).count(), [ownerKey]);
  const [showStart, setShowStart] = useState(false);
  const [kind, setKind] = useState<JourneyKind>("trip");
  const [label, setLabel] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const active = journeys?.find((journey) => journey.status === "active");
  const history = journeys?.filter((journey) => journey.status !== "active") ?? [];

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    if (!label.trim()) {
      setError("Give this journey a short name.");
      return;
    }
    setSubmitting(true);
    try {
      const journey = await startJourney(kind, label);
      void syncNow();
      router.push(`/journey/?id=${journey.localId}`);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not start the journey.");
      setSubmitting(false);
    }
  }

  if (!journeys) return <LoadingState />;

  return (
    <div className="page home-page">
      <PageHeading
        eyebrow="Your travel journal"
        title="Where will today take you?"
        description="Keep the route, stops, and little details together—even when the signal disappears."
        action={!active && (
          <button className="button primary" onClick={() => setShowStart(true)}><Plus size={18} /> Start a journey</button>
        )}
      />

      {active ? (
        <section className="active-journey-panel">
          <div className="live-label"><span /> Recording in the foreground</div>
          <div>
            <p className="eyebrow">Current {active.kind}</p>
            <h2>{active.label}</h2>
            <p>Your route and new pins are saved on this device first.</p>
          </div>
          <Link className="button light" href={`/journey/?id=${active.localId}`}>Open journey <ArrowRight size={18} /></Link>
        </section>
      ) : (
        <section className="start-hero">
          <div className="hero-map-art" aria-hidden="true">
            <span className="route-line one" />
            <span className="route-line two" />
            <span className="map-pin-art"><MapPin /></span>
          </div>
          <div>
            <span className="hero-chip"><Sparkles size={15} /> Ready when you are</span>
            <h2>Turn a day out into a story you can retrace.</h2>
            <p>Start a trip or a short outing. Pinerary will capture your foreground route while you add memorable stops.</p>
            <button className="button primary" onClick={() => setShowStart(true)}><Plus size={18} /> Start now</button>
          </div>
        </section>
      )}

      <section className="stat-strip" aria-label="Journal overview">
        <div><strong>{history.length}</strong><span>past journeys</span></div>
        <div><strong>{placesCount ?? 0}</strong><span>saved places</span></div>
        <div><strong>Motorcycle</strong><span>default road mode</span></div>
      </section>

      <section className="section-block">
        <div className="section-heading"><div><p className="eyebrow">Your archive</p><h2>Previous journeys</h2></div></div>
        {history.length ? (
          <div className="journey-grid">{history.map((journey) => <JourneyCard key={journey.localId} journey={journey} />)}</div>
        ) : (
          <EmptyState title="Your first story starts here" copy="Completed trips and outings will form a private, searchable archive." />
        )}
      </section>

      {showStart && (
        <div className="modal-backdrop" role="presentation" onMouseDown={(event) => event.target === event.currentTarget && setShowStart(false)}>
          <section className="modal-card" role="dialog" aria-modal="true" aria-labelledby="start-title">
            <button className="modal-close" onClick={() => setShowStart(false)} aria-label="Close">×</button>
            <p className="eyebrow">New journey</p>
            <h2 id="start-title">What are you starting?</h2>
            <p className="muted">Tracking works while this PWA is open. Android background tracking comes in the final Capacitor phase.</p>
            <form onSubmit={submit}>
              <div className="segmented-control" aria-label="Journey type">
                <button type="button" className={kind === "trip" ? "active" : ""} onClick={() => setKind("trip")}><Route size={18} /> Trip</button>
                <button type="button" className={kind === "outing" ? "active" : ""} onClick={() => setKind("outing")}><Bike size={18} /> Outing</button>
              </div>
              <label className="field">
                <span>Name</span>
                <input value={label} onChange={(event) => setLabel(event.target.value)} maxLength={160} placeholder={kind === "trip" ? "Goa weekend" : "Sunday ride"} autoFocus />
              </label>
              {kind === "outing" && <InlineMessage>Outings receive a reminder near 23 hours and automatically finish at 24 hours.</InlineMessage>}
              {error && <InlineMessage tone="error">{error}</InlineMessage>}
              <button className="button primary wide" disabled={submitting}>{submitting ? "Starting…" : `Start ${kind}`}</button>
            </form>
          </section>
        </div>
      )}
    </div>
  );
}
