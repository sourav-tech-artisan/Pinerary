"use client";

import { useLiveQuery } from "dexie-react-hooks";
import { Bell, BellOff, Cloud, Download, RefreshCw, Save, ShieldCheck, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useApp } from "@/components/app-provider";
import { InlineMessage, PageHeading } from "@/components/ui";
import { deleteDevice, getProfile, registerDevice, updateProfile } from "@/lib/api/client";
import {
  getAPIBaseURL,
  getAuthToken,
  getInstallationID,
  getOwnerKey,
  getPreferredTransportMode,
  saveClientConfig,
  savePreferredTransportMode,
} from "@/lib/config";
import { db } from "@/lib/db";
import type { TransportMode, UserDTO } from "@/lib/types";

const PUSH_DEVICE_KEY = "pinerary.push-device-id";

function vapidKey(value: string): Uint8Array<ArrayBuffer> {
  const padding = "=".repeat((4 - (value.length % 4)) % 4);
  const raw = atob((value + padding).replace(/-/g, "+").replace(/_/g, "/"));
  const output = new Uint8Array(new ArrayBuffer(raw.length));
  for (let index = 0; index < raw.length; index += 1) output[index] = raw.charCodeAt(index);
  return output;
}

export default function SettingsPage() {
  const app = useApp();
  const ownerKey = getOwnerKey();
  const pending = useLiveQuery(() => db.outbox.where("ownerKey").equals(ownerKey).count(), [ownerKey]) ?? 0;
  const [apiURL, setAPIURL] = useState("");
  const [token, setToken] = useState("");
  const [profile, setProfile] = useState<UserDTO>();
  const [displayName, setDisplayName] = useState("");
  const [mode, setMode] = useState<TransportMode>("motorcycle");
  const [pushEnabled, setPushEnabled] = useState(false);
  const [message, setMessage] = useState<{ text: string; tone: "info" | "error" | "success" }>();

  useEffect(() => {
    queueMicrotask(() => {
      setAPIURL(getAPIBaseURL());
      setToken(getAuthToken());
      setMode(getPreferredTransportMode());
    });
    if ("serviceWorker" in navigator) {
      navigator.serviceWorker.ready
        .then((registration) => registration.pushManager?.getSubscription())
        .then((subscription) => setPushEnabled(Boolean(subscription)))
        .catch(() => undefined);
    }
    if (navigator.onLine) {
      getProfile().then((user) => {
        setProfile(user);
        setDisplayName(user.display_name);
        setMode(user.default_transport_mode);
        savePreferredTransportMode(user.default_transport_mode);
      }).catch(() => undefined);
    }
  }, []);

  function saveConnection(event: React.FormEvent) {
    event.preventDefault();
    if (!apiURL.trim() || !token.trim()) return;
    saveClientConfig(apiURL, token);
    window.location.reload();
  }

  async function saveProfile(event: React.FormEvent) {
    event.preventDefault();
    try {
      const user = await updateProfile(displayName, mode);
      setProfile(user);
      savePreferredTransportMode(user.default_transport_mode);
      setMessage({ text: "Profile preferences saved.", tone: "success" });
    } catch (reason) {
      setMessage({ text: reason instanceof Error ? reason.message : "Could not save your profile.", tone: "error" });
    }
  }

  async function enablePush() {
    const publicKey = process.env.NEXT_PUBLIC_VAPID_PUBLIC_KEY;
    if (!publicKey) {
      setMessage({ text: "Add NEXT_PUBLIC_VAPID_PUBLIC_KEY before enabling notifications.", tone: "info" });
      return;
    }
    try {
      const permission = await Notification.requestPermission();
      if (permission !== "granted") throw new Error("Notification permission was not granted.");
      const registration = await navigator.serviceWorker.ready;
      const subscription = await registration.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: vapidKey(publicKey) });
      const device = await registerDevice(getInstallationID(), subscription.toJSON() as Record<string, unknown>);
      window.localStorage.setItem(PUSH_DEVICE_KEY, device.id);
      setPushEnabled(true);
      setMessage({ text: "Journey reminders are enabled on this device.", tone: "success" });
    } catch (reason) {
      setMessage({ text: reason instanceof Error ? reason.message : "Could not enable notifications.", tone: "error" });
    }
  }

  async function disablePush() {
    const registration = await navigator.serviceWorker.ready;
    await (await registration.pushManager.getSubscription())?.unsubscribe();
    const deviceID = window.localStorage.getItem(PUSH_DEVICE_KEY);
    if (deviceID) await deleteDevice(deviceID).catch(() => undefined);
    window.localStorage.removeItem(PUSH_DEVICE_KEY);
    setPushEnabled(false);
  }

  async function resetLocalData() {
    if (!window.confirm("Delete all offline Pinerary data on this device? Synced server data is not deleted.")) return;
    await db.delete();
    window.location.reload();
  }

  return (
    <div className="page settings-page">
      <PageHeading eyebrow="App preferences" title="Settings" description="Control synchronization, your default travel mode, and this device." />
      {message && <InlineMessage tone={message.tone}>{message.text}</InlineMessage>}

      <div className="settings-grid">
        <section className="settings-card">
          <div className="settings-icon"><Cloud /></div>
          <div className="settings-copy"><h2>Synchronization</h2><p>{pending} queued change{pending === 1 ? "" : "s"}. Last sync {app.lastSync ? app.lastSync.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) : "not completed yet"}.</p></div>
          <button className="button secondary" onClick={app.syncNow} disabled={app.syncState === "syncing"}><RefreshCw size={17} className={app.syncState === "syncing" ? "spin" : ""} /> Sync now</button>
        </section>

        <section className="settings-card form-card">
          <div className="settings-icon"><ShieldCheck /></div>
          <div className="settings-copy">
            <h2>Backend connection</h2>
            <p>Development mode accepts any non-empty bearer token. Production will use the selected OIDC provider.</p>
            <form onSubmit={saveConnection}>
              <label className="field"><span>Go API URL</span><input value={apiURL} onChange={(event) => setAPIURL(event.target.value)} inputMode="url" /></label>
              <label className="field"><span>Development token</span><input value={token} onChange={(event) => setToken(event.target.value)} autoComplete="off" /></label>
              <button className="button secondary"><Save size={17} /> Save and reload</button>
            </form>
          </div>
        </section>

        <section className="settings-card form-card">
          <div className="settings-icon"><Save /></div>
          <div className="settings-copy">
            <h2>Your profile</h2>
            <p>{profile ? `Connected as ${profile.subject}` : "Connect to edit server preferences."}</p>
            <form onSubmit={saveProfile}>
              <label className="field"><span>Display name</span><input value={displayName} maxLength={160} onChange={(event) => setDisplayName(event.target.value)} /></label>
              <label className="field"><span>Default road mode</span><select value={mode} onChange={(event) => setMode(event.target.value as TransportMode)}><option value="motorcycle">Motorcycle</option><option value="car">Car</option><option value="walking">Walking</option></select></label>
              <button className="button secondary" disabled={!profile}><Save size={17} /> Save profile</button>
            </form>
          </div>
        </section>

        <section className="settings-card">
          <div className="settings-icon">{pushEnabled ? <Bell /> : <BellOff />}</div>
          <div className="settings-copy"><h2>Outing reminders</h2><p>Web Push support depends on the browser. On iOS, install the PWA to the home screen first.</p></div>
          <button className="button secondary" onClick={pushEnabled ? disablePush : enablePush}>{pushEnabled ? <BellOff size={17} /> : <Bell size={17} />} {pushEnabled ? "Disable" : "Enable"}</button>
        </section>

        <section className="settings-card">
          <div className="settings-icon"><Download /></div>
          <div className="settings-copy"><h2>Install Pinerary</h2><p>Use your browser’s “Add to Home Screen” action. HTTPS is required outside localhost.</p></div>
        </section>

        <section className="settings-card danger-card">
          <div className="settings-icon"><Trash2 /></div>
          <div className="settings-copy"><h2>Reset offline data</h2><p>Removes IndexedDB data and unsynchronized changes from this device only.</p></div>
          <button className="button danger-quiet" onClick={resetLocalData}><Trash2 size={17} /> Reset device</button>
        </section>
      </div>
    </div>
  );
}
