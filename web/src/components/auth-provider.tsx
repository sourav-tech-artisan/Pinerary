"use client";

import { Auth0Provider, useAuth0, type AppState, type User } from "@auth0/auth0-react";
import { LoaderCircle, LogIn, MapPinned, UserPlus } from "lucide-react";
import { usePathname, useRouter } from "next/navigation";
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { clearAccessTokenProvider, installAccessTokenProvider } from "@/lib/auth-session";
import {
  clearAuthenticatedOwner,
  getAuth0Config,
  getAuthMode,
  setAuthenticatedOwner,
} from "@/lib/config";

interface SessionUser {
  subject: string;
  name?: string;
  email?: string;
  picture?: string;
}

interface AuthContextValue {
  mode: "auth0" | "development";
  user?: SessionUser;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

function AuthLoading({ message = "Preparing your trips…" }: { message?: string }) {
  return (
    <main className="auth-screen" aria-busy="true">
      <div className="auth-card auth-loading">
        <span className="auth-mark" aria-hidden="true"><MapPinned /></span>
        <LoaderCircle className="spin" aria-hidden="true" />
        <p>{message}</p>
      </div>
    </main>
  );
}

function AuthWelcome({
  login,
  signup,
}: {
  login: () => Promise<void>;
  signup: () => Promise<void>;
}) {
  return (
    <main className="auth-screen">
      <section className="auth-card">
        <span className="auth-mark" aria-hidden="true"><MapPinned /></span>
        <p className="eyebrow">Your private travel journal</p>
        <h1>Keep every road, stop, and story.</h1>
        <p className="auth-intro">Sign in to capture trips, save places, and keep your offline changes synchronized across devices.</p>
        <div className="auth-actions">
          <button className="button primary" onClick={() => void login()}><LogIn size={18} /> Log in</button>
          <button className="button secondary" onClick={() => void signup()}><UserPlus size={18} /> Create account</button>
        </div>
      </section>
    </main>
  );
}

function AuthFailure({ message, retry }: { message: string; retry: () => Promise<void> }) {
  return (
    <main className="auth-screen">
      <section className="auth-card">
        <span className="auth-mark" aria-hidden="true"><MapPinned /></span>
        <p className="eyebrow">Authentication problem</p>
        <h1>We could not sign you in.</h1>
        <p className="auth-intro">{message}</p>
        <button className="button primary" onClick={() => void retry()}><LogIn size={18} /> Try again</button>
      </section>
    </main>
  );
}

function sessionUser(user: User): SessionUser | undefined {
  if (!user.sub) return undefined;
  return {
    subject: user.sub,
    name: user.name,
    email: user.email,
    picture: user.picture,
  };
}

function ReadyAuthenticatedSession({
  children,
  currentUser,
  getToken,
  logout,
}: {
  children: React.ReactNode;
  currentUser: SessionUser;
  getToken: () => Promise<string | undefined>;
  logout: () => Promise<void>;
}) {
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let active = true;
    setAuthenticatedOwner(currentUser.subject);
    const uninstall = installAccessTokenProvider(async () => {
      const token = await getToken();
      if (!token) throw new Error("Auth0 did not return an API access token. Please sign in again.");
      return token;
    });
    queueMicrotask(() => {
      if (active) setReady(true);
    });
    return () => {
      active = false;
      uninstall();
    };
  }, [currentUser.subject, getToken]);

  const value = useMemo<AuthContextValue>(
    () => ({ mode: "auth0", user: currentUser, logout }),
    [currentUser, logout],
  );

  if (!ready) return <AuthLoading message="Opening your travel journal…" />;
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

function AuthenticatedSession({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const {
    error,
    getAccessTokenSilently,
    isAuthenticated,
    isLoading,
    loginWithRedirect,
    logout: auth0Logout,
    user,
  } = useAuth0();

  useEffect(() => {
    if (!isAuthenticated) {
      clearAccessTokenProvider();
      clearAuthenticatedOwner();
    }
  }, [isAuthenticated]);

  const login = useCallback(
    () => loginWithRedirect({ appState: { returnTo: pathname || "/" } }),
    [loginWithRedirect, pathname],
  );
  const signup = useCallback(
    () => loginWithRedirect({
      appState: { returnTo: pathname || "/" },
      authorizationParams: { screen_hint: "signup" },
    }),
    [loginWithRedirect, pathname],
  );
  const retry = useCallback(
    () => loginWithRedirect({
      appState: { returnTo: pathname || "/" },
      authorizationParams: { prompt: "login" },
    }),
    [loginWithRedirect, pathname],
  );
  const logout = useCallback(async () => {
    clearAccessTokenProvider();
    clearAuthenticatedOwner();
    await auth0Logout({ logoutParams: { returnTo: window.location.origin } });
  }, [auth0Logout]);

  if (isLoading) return <AuthLoading />;
  if (error) return <AuthFailure message={error.message} retry={retry} />;
  if (!isAuthenticated) return <AuthWelcome login={login} signup={signup} />;
  const currentUser = user ? sessionUser(user) : undefined;
  if (!currentUser) return <AuthFailure message="Auth0 returned a user without a subject identifier." retry={retry} />;

  return (
    <ReadyAuthenticatedSession currentUser={currentUser} getToken={getAccessTokenSilently} logout={logout}>
      {children}
    </ReadyAuthenticatedSession>
  );
}

function Auth0SessionProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [origin, setOrigin] = useState<string>();
  const config = getAuth0Config();

  useEffect(() => {
    queueMicrotask(() => setOrigin(window.location.origin));
  }, []);

  const onRedirectCallback = useCallback(
    (appState?: AppState) => router.replace(appState?.returnTo || "/"),
    [router],
  );

  if (!config.domain || !config.clientId || !config.audience) {
    return (
      <AuthFailure
        message="Auth0 is not configured for this build. Add the three NEXT_PUBLIC_AUTH0_* values and rebuild the PWA."
        retry={async () => window.location.reload()}
      />
    );
  }
  if (!origin) return <AuthLoading />;

  return (
    <Auth0Provider
      domain={config.domain}
      clientId={config.clientId}
      authorizationParams={{
        audience: config.audience,
        redirect_uri: origin,
        scope: "openid profile email offline_access",
      }}
      cacheLocation="localstorage"
      useRefreshTokens
      onRedirectCallback={onRedirectCallback}
    >
      <AuthenticatedSession>{children}</AuthenticatedSession>
    </Auth0Provider>
  );
}

function DevelopmentSessionProvider({ children }: { children: React.ReactNode }) {
  const value = useMemo<AuthContextValue>(
    () => ({ mode: "development", logout: async () => undefined }),
    [],
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function PineraryAuthProvider({ children }: { children: React.ReactNode }) {
  if (getAuthMode() === "development") {
    return <DevelopmentSessionProvider>{children}</DevelopmentSessionProvider>;
  }
  return <Auth0SessionProvider>{children}</Auth0SessionProvider>;
}

export function usePineraryAuth(): AuthContextValue {
  const value = useContext(AuthContext);
  if (!value) throw new Error("usePineraryAuth must be used inside PineraryAuthProvider");
  return value;
}
