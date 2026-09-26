const TOKEN_KEY = "pinerary.auth-token";
const API_URL_KEY = "pinerary.api-url";
const INSTALLATION_KEY = "pinerary.installation-id";
const TRANSPORT_MODE_KEY = "pinerary.transport-mode";
const AUTHENTICATED_OWNER_KEY = "pinerary.authenticated-owner";

const defaultAPIURL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api/v1";
const defaultDevelopmentToken = process.env.NEXT_PUBLIC_DEFAULT_DEV_TOKEN ?? "alice";

function browserValue(key: string, fallback: string): string {
  if (typeof window === "undefined") return fallback;
  return window.localStorage.getItem(key)?.trim() || fallback;
}

export function getAPIBaseURL(): string {
  if (getAuthMode() === "auth0") return defaultAPIURL.replace(/\/+$/, "");
  return browserValue(API_URL_KEY, defaultAPIURL).replace(/\/+$/, "");
}

export function getAuthMode(): "auth0" | "development" {
  return process.env.NEXT_PUBLIC_AUTH_MODE === "development" ? "development" : "auth0";
}

export function getAuth0Config() {
  return {
    domain: process.env.NEXT_PUBLIC_AUTH0_DOMAIN?.trim() ?? "",
    clientId: process.env.NEXT_PUBLIC_AUTH0_CLIENT_ID?.trim() ?? "",
    audience: process.env.NEXT_PUBLIC_AUTH0_AUDIENCE?.trim() ?? "",
  };
}

export function getDevelopmentToken(): string {
  return browserValue(TOKEN_KEY, defaultDevelopmentToken);
}

export function saveClientConfig(apiURL: string, token?: string): void {
  if (getAuthMode() !== "development") return;
  window.localStorage.setItem(API_URL_KEY, apiURL.trim().replace(/\/+$/, ""));
  if (token !== undefined) window.localStorage.setItem(TOKEN_KEY, token.trim());
}

function ownerHash(source: string): string {
  let left = 0x811c9dc5;
  let right = 0x9e3779b9;
  for (let index = 0; index < source.length; index += 1) {
    const code = source.charCodeAt(index);
    left = Math.imul(left ^ code, 0x01000193);
    right = Math.imul(right ^ code, 0x85ebca6b);
  }
  return `account-${(left >>> 0).toString(16)}${(right >>> 0).toString(16)}`;
}

export function setAuthenticatedOwner(subject: string): void {
  window.localStorage.setItem(AUTHENTICATED_OWNER_KEY, ownerHash(subject));
}

export function clearAuthenticatedOwner(): void {
  if (typeof window !== "undefined") window.localStorage.removeItem(AUTHENTICATED_OWNER_KEY);
}

export function getOwnerKey(): string {
  if (getAuthMode() === "development") return ownerHash(getDevelopmentToken());
  return browserValue(AUTHENTICATED_OWNER_KEY, "account-authentication-pending");
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
  authenticatedOwner: AUTHENTICATED_OWNER_KEY,
} as const;
