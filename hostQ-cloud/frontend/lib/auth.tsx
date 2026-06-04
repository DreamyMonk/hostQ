"use client";

import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { apiJSON, setAccessToken } from "./api";

export type User = {
  id: string;
  email: string;
  role: "superadmin" | "admin" | "tenant";
  displayName?: string;
};

type AuthState = {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<User>;
  logout: () => Promise<void>;
};

const AuthCtx = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  // Bootstrap: try to refresh the session, then fetch /me.
  useEffect(() => {
    (async () => {
      try {
        const r = await fetch("/api/auth/refresh", { method: "POST", credentials: "include" });
        if (r.ok) {
          const { accessToken } = (await r.json()) as { accessToken: string };
          setAccessToken(accessToken);
          const me = await apiJSON<User>("/api/auth/me");
          setUser(me);
        }
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  async function login(email: string, password: string): Promise<User> {
    const res = await apiJSON<{ accessToken: string; user: User }>("/api/auth/login", {
      method: "POST",
      json: { email, password },
    });
    setAccessToken(res.accessToken);
    setUser(res.user);
    return res.user;
  }

  async function logout() {
    try {
      await apiJSON("/api/auth/logout", { method: "POST" });
    } catch {}
    setAccessToken(null);
    setUser(null);
  }

  return <AuthCtx.Provider value={{ user, loading, login, logout }}>{children}</AuthCtx.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthCtx);
  if (!ctx) throw new Error("useAuth outside AuthProvider");
  return ctx;
}
