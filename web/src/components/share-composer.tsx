"use client";

import { ArrowDown, ArrowUp, Check, Copy, ExternalLink, Share2 } from "lucide-react";
import { useMemo, useState } from "react";
import { createShare } from "@/lib/api/client";
import type { LocalPhoto, LocalStop } from "@/lib/types";

interface DraftItem {
  stop: LocalStop;
  name: string;
  note: string;
  included: boolean;
  photoIDs: string[];
}

export function ShareComposer({
  journeyID,
  defaultTitle,
  stops,
  photos,
  onClose,
}: {
  journeyID: string;
  defaultTitle: string;
  stops: LocalStop[];
  photos: LocalPhoto[];
  onClose: () => void;
}) {
  const initial = useMemo<DraftItem[]>(
    () => stops.map((stop) => ({ stop, name: stop.name, note: stop.note, included: true, photoIDs: [] })),
    [stops],
  );
  const [title, setTitle] = useState(defaultTitle);
  const [items, setItems] = useState(initial);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [result, setResult] = useState<{ url: string; message: string }>();

  function update(index: number, changes: Partial<DraftItem>) {
    setItems((current) => current.map((item, itemIndex) => itemIndex === index ? { ...item, ...changes } : item));
  }

  function move(index: number, offset: number) {
    setItems((current) => {
      const target = index + offset;
      if (target < 0 || target >= current.length) return current;
      const next = [...current];
      [next[index], next[target]] = [next[target], next[index]];
      return next;
    });
  }

  async function publish() {
    setBusy(true);
    setError("");
    try {
      const share = await createShare(journeyID, {
        title: title.trim(),
        items: items.filter((item) => item.included).map((item) => ({
          stop_id: item.stop.serverId!,
          display_name: item.name,
          note: item.note,
          photo_ids: item.photoIDs,
        })),
      });
      setResult({ url: share.url, message: share.message });
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not create the public itinerary.");
    } finally {
      setBusy(false);
    }
  }

  async function shareResult() {
    if (!result) return;
    if (navigator.share) {
      await navigator.share({ title, text: result.message }).catch(() => undefined);
      return;
    }
    await navigator.clipboard.writeText(result.message);
  }

  return (
    <div className="modal-backdrop" role="presentation" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <section className="modal-card share-modal" role="dialog" aria-modal="true" aria-labelledby="share-title">
        <button className="modal-close" onClick={onClose} aria-label="Close">×</button>
        {!result ? (
          <>
            <p className="eyebrow">Share a snapshot</p>
            <h2 id="share-title">Arrange the public itinerary</h2>
            <p className="muted">Changes here affect only this link. Your original timeline stays untouched.</p>
            <label className="field"><span>Title</span><input value={title} maxLength={160} onChange={(event) => setTitle(event.target.value)} /></label>
            <div className="share-items">
              {items.map((item, index) => {
                const availablePhotos = photos.filter((photo) => photo.stopLocalId === item.stop.localId && photo.serverId && photo.thumbnailURL);
                return (
                  <article className={item.included ? "share-item" : "share-item excluded"} key={item.stop.localId}>
                    <label className="check-row">
                      <input type="checkbox" checked={item.included} onChange={(event) => update(index, { included: event.target.checked })} />
                      <span>{index + 1}</span>
                    </label>
                    <div className="share-item-fields">
                      <input aria-label="Shared stop name" value={item.name} maxLength={200} onChange={(event) => update(index, { name: event.target.value })} />
                      <textarea aria-label="Shared stop note" value={item.note} maxLength={5000} rows={2} onChange={(event) => update(index, { note: event.target.value })} />
                      {availablePhotos.length > 0 && (
                        <div className="share-photo-options">
                          {availablePhotos.map((photo) => (
                            <label key={photo.localId}>
                              <input
                                type="checkbox"
                                checked={photo.serverId ? item.photoIDs.includes(photo.serverId) : false}
                                onChange={(event) => {
                                  if (!photo.serverId) return;
                                  update(index, {
                                    photoIDs: event.target.checked
                                      ? [...item.photoIDs, photo.serverId]
                                      : item.photoIDs.filter((id) => id !== photo.serverId),
                                  });
                                }}
                              /> Photo
                            </label>
                          ))}
                        </div>
                      )}
                    </div>
                    <div className="reorder-buttons">
                      <button onClick={() => move(index, -1)} disabled={index === 0} aria-label="Move stop up"><ArrowUp size={16} /></button>
                      <button onClick={() => move(index, 1)} disabled={index === items.length - 1} aria-label="Move stop down"><ArrowDown size={16} /></button>
                    </div>
                  </article>
                );
              })}
            </div>
            {error && <div className="inline-message error">{error}</div>}
            <button className="button primary wide" onClick={publish} disabled={busy || !title.trim() || !items.some((item) => item.included && item.stop.serverId)}>
              <Share2 size={18} /> {busy ? "Creating link…" : "Create public link"}
            </button>
          </>
        ) : (
          <div className="share-success">
            <span className="success-mark"><Check /></span>
            <p className="eyebrow">Itinerary ready</p>
            <h2 id="share-title">Your private link is live</h2>
            <p>Anyone with this unlisted link can view the selected snapshot. You can revoke it through the API.</p>
            <a className="share-url" href={result.url} target="_blank" rel="noreferrer">{result.url} <ExternalLink size={15} /></a>
            <div className="button-row">
              <button className="button primary" onClick={shareResult}><Share2 size={18} /> Share</button>
              <button className="button secondary" onClick={() => navigator.clipboard.writeText(result.message)}><Copy size={18} /> Copy text</button>
            </div>
            <a className="whatsapp-link" href={`https://wa.me/?text=${encodeURIComponent(result.message)}`} target="_blank" rel="noreferrer">Open in WhatsApp</a>
          </div>
        )}
      </section>
    </div>
  );
}
