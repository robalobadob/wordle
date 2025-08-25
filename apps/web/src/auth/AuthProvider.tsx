/**
 * AuthProvider.tsx
 *
 * React context for authentication state and actions.
 * - Persists session via server cookies (fetch wrappers include credentials).
 * - Exposes `me`, `loading`, and auth mutators (`signup`, `login`, `logout`, `refresh`).
 * - On mount, calls `refresh()` to resolve current user (guest vs. authenticated).
 *
 * Notes:
 * - `signup`/`login`/`logout` delegate to API endpoints and then call `refresh`
 *   to sync client state. After signup/login, the server may also "claim" any
 *   anonymous/guest games associated with the session.
 */

import React, { createContext, useContext, useEffect, useState } from 'react';
import { apiGET, apiPOST } from '../lib/api';

/** Minimal user shape returned by the API when authenticated; `null` when guest. */
type Me = { id: string; username: string } | null;

/** Public interface of the auth context. */
type AuthCtx = {
  /** Current user object (or `null` if guest). */
  me: Me;
  /** True until the initial `refresh()` completes (or while an explicit refresh is running). */
  loading: boolean;

  /** Create a new account, then refresh local auth state. */
  signup: (u: string, p: string) => Promise<void>;
  /** Log in with credentials, then refresh local auth state. */
  login: (u: string, p: string) => Promise<void>;
  /** Log out current session, then refresh local auth state. */
  logout: () => Promise<void>;
  /** Query `/auth/me` to synchronize `me` from the server. */
  refresh: () => Promise<void>;
};

/** Internal context instance (initialized non-null, provided by <AuthProvider/>). */
const Ctx = createContext<AuthCtx>(null!);

/**
 * Top-level provider for authentication state.
 *
 * Usage:
 * ```tsx
 * <AuthProvider>
 *   <App />
 * </AuthProvider>
 * ```
 * In components:
 * ```tsx
 * const { me, loading, login, logout } = useAuth();
 * ```
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [me, setMe] = useState<Me>(null);
  const [loading, setLoading] = useState(true);

  /**
   * Fetch current user from the server.
   * - On success: set `me` to the user object.
   * - On failure (e.g., 401): treat as guest (`me = null`).
   * Always clears `loading` at the end.
   */
  const refresh = async () => {
    try {
      const m = await apiGET<Me>('/auth/me');
      setMe(m);
    } catch {
      setMe(null); // guest
    } finally {
      setLoading(false);
    }
  };

  // Resolve auth state on first mount.
  useEffect(() => {
    refresh();
  }, []);

  /**
   * Create account then refresh. Server may merge anonymous progress into account.
   */
  const signup = async (u: string, p: string) => {
    await apiPOST('/auth/signup', { username: u, password: p });
    await refresh(); // claims guest games on server
  };

  /** Log in then refresh. */
  const login = async (u: string, p: string) => {
    await apiPOST('/auth/login', { username: u, password: p });
    await refresh();
  };

  /** Log out then refresh (sets `me` to null). */
  const logout = async () => {
    await apiPOST('/auth/logout');
    await refresh();
  };

  return (
    <Ctx.Provider value={{ me, loading, signup, login, logout, refresh }}>
      {children}
    </Ctx.Provider>
  );
}

/**
 * Hook to consume the authentication context.
 * @returns The `AuthCtx` with `me`, `loading`, and auth actions.
 */
export const useAuth = () => useContext(Ctx);
