"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Compass, House, MapPinned, Settings, WifiOff } from "lucide-react";
import { useLiveQuery } from "dexie-react-hooks";
import { useApp } from "@/components/app-provider";
import { ForegroundTracker } from "@/components/foreground-tracker";
import { getOwnerKey } from "@/lib/config";
import { db } from "@/lib/db";

const navigation = [
  { href: "/", label: "Trips", icon: House },
  { href: "/places/", label: "Places", icon: MapPinned },
  { href: "/nearby/", label: "Nearby", icon: Compass },
  { href: "/settings/", label: "Settings", icon: Settings },
];

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { connection, syncState } = useApp();
  const ownerKey = getOwnerKey();
  const activeJourney = useLiveQuery(
    () => db.journeys.where("[ownerKey+status]").equals([ownerKey, "active"]).first(),
    [ownerKey],
  );

  return (
    <div className="app-frame">
      <aside className="side-rail">
        <Link className="brand" href="/" aria-label="Pinerary home">
          <span className="brand-mark" aria-hidden="true">P</span>
          <span>
            <strong>Pinerary</strong>
            <small>Go somewhere worth keeping</small>
          </span>
        </Link>
        <nav className="primary-nav" aria-label="Primary navigation">
          {navigation.map(({ href, label, icon: Icon }) => {
            const active = href === "/" ? pathname === "/" : pathname.startsWith(href.replace(/\/$/, ""));
            return (
              <Link href={href} className={active ? "nav-link active" : "nav-link"} key={href}>
                <Icon size={20} strokeWidth={1.8} />
                <span>{label}</span>
              </Link>
            );
          })}
        </nav>
        <div className="rail-status">
          <span className={`status-dot ${connection === "offline" ? "offline" : syncState}`} />
          {connection === "offline" ? "Working offline" : syncState === "syncing" ? "Syncing changes" : "Saved locally"}
        </div>
      </aside>

      <div className="app-content">
        {connection === "offline" && (
          <div className="offline-banner" role="status">
            <WifiOff size={16} /> Offline—new captures will sync when you reconnect.
          </div>
        )}
        {activeJourney && <div className="global-tracker"><ForegroundTracker journey={activeJourney} /></div>}
        <main>{children}</main>
      </div>

      <nav className="bottom-nav" aria-label="Primary navigation">
        {navigation.map(({ href, label, icon: Icon }) => {
          const active = href === "/" ? pathname === "/" : pathname.startsWith(href.replace(/\/$/, ""));
          return (
            <Link href={href} className={active ? "active" : ""} key={href}>
              <Icon size={21} strokeWidth={active ? 2.3 : 1.7} />
              <span>{label}</span>
            </Link>
          );
        })}
      </nav>
    </div>
  );
}
