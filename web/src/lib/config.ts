const TOKEN_KEY = "pinerary.auth-token";
const API_URL_KEY = "pinerary.api-url";
const INSTALLATION_KEY = "pinerary.installation-id";
const TRANSPORT_MODE_KEY = "pinerary.transport-mode";

const defaultAPIURL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api/v1";
const defaultDevelopmentToken = process.env.NEXT_PUBLIC_DEFAULT_DEV_TOKEN ?? "alice";

function browserValue(key: string, fallback: string): string {
  if (typeof window === "undefined") return fallback;
  return window.localStorage.getItem(key)?.trim() || fallback;
}

export function getAPIBaseURL(): string {
  return browserValue(API_URL_KEY, defaultAPIURL).replace(/\/+$/, "");
}

export function getAuthToken(): string {
  return browserValue(TOKEN_KEY, defaultDevelopmentToken);
}

export function saveClientConfig(apiURL: string, token: string): void {
  window.localStorage.setItem(API_URL_KEY, apiURL.trim().replace(/\/+$/, ""));
  window.localStorage.setItem(TOKEN_KEY, token.trim());
}

export function getOwnerKey(): string {
  const source = getAuthToken();
  let left = 0x811c9dc5;
  let right = 0x9e3779b9;
  for (let index = 0; index < source.length; index += 1) {
    const code = source.charCodeAt(index);
    left = Math.imul(left ^ code, 0x01000193);
    right = Math.imul(right ^ code, 0x85ebca6b);
  }
  return `account-${(left >>> 0).toString(16)}${(right >>> 0).toString(16)}`;
}

export function getInstallationID(): string {
  if (typeof window === "undefined") return "server-render";
  let id = window.localStorage.getItem(INSTALLATION_KEY);
  if (!id) {
    id = crypto.randomUUID();
    window.localStorage.setItem(INSTALLATION_KEY, id);
  }
  return id;
}

export function getPreferredTransportMode(): "motorcycle" | "car" | "walking" {
  const value = browserValue(TRANSPORT_MODE_KEY, "motorcycle");
  return value === "car" || value === "walking" ? value : "motorcycle";
}

export function savePreferredTransportMode(mode: "motorcycle" | "car" | "walking"): void {
  window.localStorage.setItem(TRANSPORT_MODE_KEY, mode);
}

export const clientConfigKeys = {
  token: TOKEN_KEY,
  apiURL: API_URL_KEY,
  installation: INSTALLATION_KEY,
  transportMode: TRANSPORT_MODE_KEY,
} as const;
