"use client";

import { Check, Edit3, MapPin, X } from "lucide-react";
import { useState } from "react";
import { PhotoStrip } from "@/components/photo-strip";
import { editJourneyStop } from "@/lib/repository";
import type { LocalStop } from "@/lib/types";

function formatTime(value: string) {
  return new Intl.DateTimeFormat("en-IN", { hour: "numeric", minute: "2-digit" }).format(new Date(value));
}

export function TimelineStop({ stop, journeyServerID }: { stop: LocalStop; journeyServerID?: string }) {
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(stop.name);
  const [note, setNote] = useState(stop.note);

  async function save() {
    if (!name.trim()) return;
    await editJourneyStop(stop.localId, name, note);
    setEditing(false);
  }

  return (
    <article className="timeline-stop">
      <div className="timeline-marker"><MapPin size={17} /></div>
      <div className="timeline-card">
        <div className="timeline-meta"><span>Stop {stop.sequence}</span><time>{formatTime(stop.capturedAt)}</time></div>
        {editing ? (
          <div className="stop-edit-form">
            <input value={name} maxLength={200} onChange={(event) => setName(event.target.value)} aria-label="Stop name" />
            <textarea value={note} onChange={(event) => setNote(event.target.value)} rows={2} aria-label="Stop note" />
            <div className="button-row compact">
              <button className="icon-text-button" onClick={save}><Check size={15} /> Save</button>
              <button className="icon-text-button" onClick={() => setEditing(false)}><X size={15} /> Cancel</button>
            </div>
          </div>
        ) : (
          <>
            <div className="stop-title-row"><h3>{stop.name}</h3><button onClick={() => setEditing(true)} aria-label={`Edit ${stop.name}`}><Edit3 size={16} /></button></div>
            {stop.note && <p>{stop.note}</p>}
          </>
        )}
        <small className="coordinates">{stop.latitude.toFixed(5)}, {stop.longitude.toFixed(5)}</small>
        <PhotoStrip stop={stop} journeyServerID={journeyServerID} />
      </div>
    </article>
  );
}
