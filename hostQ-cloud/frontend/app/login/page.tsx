"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { Server } from "lucide-react";

export default function LoginPage() {
  const { login } = useAuth();
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    setBusy(true);
    try {
      const u = await login(email, password);
      router.replace(u.role === "tenant" ? "/user" : "/admin");
    } catch (e: any) {
      setErr(e?.message || "Sign in failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="min-h-screen grid place-items-center p-6 bg-canvas">
      <div className="w-full max-w-sm">
        <div className="flex items-center justify-center mb-8">
          <div className="flex items-center gap-2">
            <div className="grid h-9 w-9 place-items-center rounded-lg bg-ink text-surface">
              <Server className="h-4 w-4" />
            </div>
            <div>
              <div className="text-base font-semibold tracking-tight">hostQ-cloud</div>
              <div className="text-[11px] text-muted -mt-0.5">control panel</div>
            </div>
          </div>
        </div>
        <Card>
          <CardContent className="pt-5">
            <form onSubmit={onSubmit} className="space-y-3">
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted">Email</label>
                <Input type="email" autoFocus required value={email} onChange={(e) => setEmail(e.target.value)} />
              </div>
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted">Password</label>
                <Input type="password" required value={password} onChange={(e) => setPassword(e.target.value)} />
              </div>
              {err && (
                <div className="rounded-md border border-danger/30 bg-danger/5 px-3 py-2 text-xs text-danger">{err}</div>
              )}
              <Button variant="primary" size="lg" className="w-full" disabled={busy}>
                {busy ? "Signing in…" : "Sign in"}
              </Button>
            </form>
          </CardContent>
        </Card>
        <div className="text-center text-xs text-faint mt-6">
          hostQ-cloud &middot; multi-tenant edition
        </div>
      </div>
    </div>
  );
}
