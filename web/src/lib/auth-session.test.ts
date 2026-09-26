import { afterEach, describe, expect, it, vi } from "vitest";
import {
  clearAccessTokenProvider,
  getAccessToken,
  installAccessTokenProvider,
} from "@/lib/auth-session";

describe("Auth0 access-token bridge", () => {
  afterEach(() => {
    clearAccessTokenProvider();
    vi.unstubAllEnvs();
  });

  it("supplies the current SDK access token to non-React API modules", async () => {
    vi.stubEnv("NEXT_PUBLIC_AUTH_MODE", "auth0");
    const provider = vi.fn().mockResolvedValue("short-lived-access-token");
    installAccessTokenProvider(provider);

    await expect(getAccessToken()).resolves.toBe("short-lived-access-token");
    expect(provider).toHaveBeenCalledOnce();
  });

  it("fails closed before an Auth0 session is ready", async () => {
    vi.stubEnv("NEXT_PUBLIC_AUTH_MODE", "auth0");
    await expect(getAccessToken()).rejects.toThrow("Authentication is not ready");
  });
});
