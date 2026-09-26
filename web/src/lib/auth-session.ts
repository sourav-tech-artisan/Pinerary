import { getAuthMode, getDevelopmentToken } from "@/lib/config";

type AccessTokenProvider = () => Promise<string>;

let accessTokenProvider: AccessTokenProvider | undefined;

export function installAccessTokenProvider(provider: AccessTokenProvider): () => void {
  accessTokenProvider = provider;
  return () => {
    if (accessTokenProvider === provider) accessTokenProvider = undefined;
  };
}

export function clearAccessTokenProvider(): void {
  accessTokenProvider = undefined;
}

export async function getAccessToken(): Promise<string> {
  if (getAuthMode() === "development") return getDevelopmentToken();
  if (!accessTokenProvider) throw new Error("Authentication is not ready. Please sign in again.");
  return accessTokenProvider();
}
