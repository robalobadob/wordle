import React, { createContext, useContext, useEffect, useState } from 'react';
import { apiGET, apiPOST } from '../lib/api';

type Me = { id: string; username: string } | null;

type AuthCtx = {
  me: Me;
  loading: boolean;
  signup: (u: string, p: string) => Promise<void>;
  login: (u: string, p: string) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
};

const Ctx = createContext<AuthCtx>(null!);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [me, setMe] = useState<Me>(null);
  const [loading, setLoading] = useState(true);

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

  useEffect(() => {
    refresh();
  }, []);

  const signup = async (u: string, p: string) => {
    await apiPOST('/auth/signup', { username: u, password: p });
    await refresh(); // claims guest games on server
  };
  const login = async (u: string, p: string) => {
    await apiPOST('/auth/login', { username: u, password: p });
    await refresh();
  };
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
export const useAuth = () => useContext(Ctx);
