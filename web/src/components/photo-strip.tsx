"use client";

import { useLiveQuery } from "dexie-react-hooks";
import { Camera, Cloud, Image as ImageIcon } from "lucide-react";
import Image from "next/image";
import { useEffect, useMemo, useState } from "react";
import { listStopPhotos } from "@/lib/api/client";
import { db } from "@/lib/db";
import { newID } from "@/lib/id";
import { queuePhoto } from "@/lib/repository";
import type { LocalPhoto, LocalStop } from "@/lib/types";

function LocalThumbnail({ photo }: { photo: LocalPhoto }) {
  const source = useMemo(() => photo.blob ? URL.createObjectURL(photo.blob) : photo.thumbnailURL, [photo.blob, photo.thumbnailURL]);
  useEffect(() => {
    return () => {
      if (photo.blob && source) URL.revokeObjectURL(source);
    };
  }, [photo.blob, source]);
  return source ? <Image src={source} alt="Captured at this stop" width={76} height={66} unoptimized /> : <ImageIcon aria-label="Photo processing" />;
}

export function PhotoStrip({ stop, journeyServerID }: { stop: LocalStop; journeyServerID?: string }) {
  const storedPhotos = useLiveQuery(() => db.photos.where("stopLocalId").equals(stop.localId).toArray(), [stop.localId]);
  const photos = useMemo(() => storedPhotos ?? [], [storedPhotos]);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!journeyServerID || !stop.serverId || !navigator.onLine) return;
    const refresh = () => listStopPhotos(journeyServerID, stop.serverId!)
      .then(async (remote) => {
        for (const photo of remote) {
          const local = await db.photos.where("serverId").equals(photo.photo_id).first();
          if (local) {
            await db.photos.update(local.localId, { thumbnailURL: photo.url, syncState: "synced" });
          } else {
            await db.photos.add({
              localId: newID(),
              ownerKey: stop.ownerKey,
              serverId: photo.photo_id,
              journeyLocalId: stop.journeyLocalId,
              stopLocalId: stop.localId,
              contentType: "image/jpeg",
              capturedAt: photo.captured_at ?? new Date().toISOString(),
              thumbnailURL: photo.url,
              syncState: "synced",
            });
          }
        }
      })
      .catch(() => undefined);
    void refresh();
    const waiting = photos.some((photo) => photo.serverId && !photo.thumbnailURL);
    if (!waiting) return;
    const interval = window.setInterval(refresh, 8_000);
    return () => window.clearInterval(interval);
  }, [journeyServerID, stop, photos]);

  async function selectPhoto(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    setError("");
    try {
      await queuePhoto(stop.journeyLocalId, stop.localId, file);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not queue the photo.");
    }
    event.target.value = "";
  }

  return (
    <div className="photo-strip">
      {photos.map((photo) => (
        <div className="photo-thumb" key={photo.localId} title={photo.syncState === "synced" ? "Uploaded" : "Waiting to upload"}>
          <LocalThumbnail photo={photo} />
          {photo.syncState !== "synced" && <span><Cloud size={12} /></span>}
        </div>
      ))}
      <label className="add-photo">
        <Camera size={18} />
        <span>Add photo</span>
        <input type="file" accept="image/jpeg,image/png,image/webp" capture="environment" onChange={selectPhoto} />
      </label>
      {error && <small className="field-error">{error}</small>}
    </div>
  );
}
