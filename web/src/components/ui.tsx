import { LoaderCircle, MapPin } from "lucide-react";

export function PageHeading({
  eyebrow,
  title,
  description,
  action,
}: {
  eyebrow?: string;
  title: string;
  description?: string;
  action?: React.ReactNode;
}) {
  return (
    <header className="page-heading">
      <div>
        {eyebrow && <p className="eyebrow">{eyebrow}</p>}
        <h1>{title}</h1>
        {description && <p>{description}</p>}
      </div>
      {action && <div className="page-action">{action}</div>}
    </header>
  );
}

export function EmptyState({ title, copy, action }: { title: string; copy: string; action?: React.ReactNode }) {
  return (
    <div className="empty-state">
      <span className="empty-icon"><MapPin size={25} /></span>
      <h2>{title}</h2>
      <p>{copy}</p>
      {action}
    </div>
  );
}

export function LoadingState({ label = "Loading your travel journal…" }: { label?: string }) {
  return (
    <div className="loading-state" role="status">
      <LoaderCircle className="spin" size={22} /> {label}
    </div>
  );
}

export function SyncBadge({ state }: { state: "pending" | "synced" | "failed" }) {
  return <span className={`sync-badge ${state}`}>{state === "synced" ? "Synced" : state === "failed" ? "Needs attention" : "Waiting to sync"}</span>;
}

export function InlineMessage({ children, tone = "info" }: { children: React.ReactNode; tone?: "info" | "error" | "success" }) {
  return <div className={`inline-message ${tone}`} role={tone === "error" ? "alert" : "status"}>{children}</div>;
}
