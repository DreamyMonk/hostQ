"use client";

import { useAuth } from "@/lib/auth";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

// Root route: bounce to the right scope based on role, or to login.
export default function Index() {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (loading) return;
    if (!user) router.replace("/login");
    else if (user.role === "tenant") router.replace("/user");
    else router.replace("/admin");
  }, [user, loading, router]);

  return (
    <div className="grid min-h-screen place-items-center">
      <div className="text-sm text-muted">Loading…</div>
    </div>
  );
}
