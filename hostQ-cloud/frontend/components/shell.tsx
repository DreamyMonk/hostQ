"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard, Globe, Folder, Database, Shield, Activity, Users, ServerCog,
  LogOut, Search, Command, Sun, Moon, ChevronRight, Server, type LucideIcon,
} from "lucide-react";
import { useTheme } from "next-themes";
import { useEffect, useState } from "react";

export type NavItem = {
  href: string;
  label: string;
  icon: LucideIcon;
  match?: (path: string) => boolean;
};

export type NavGroup = {
  title?: string;
  items: NavItem[];
};

export function Shell({
  groups,
  scope,
  children,
}: {
  groups: NavGroup[];
  scope: "admin" | "user";
  children: React.ReactNode;
}) {
  const { user, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const { theme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);

  // Auth gate: while bootstrapping, render nothing. If the role doesn't fit
  // this scope, bounce.
  useEffect(() => {
    if (!user) return;
    if (scope === "admin" && user.role === "tenant") router.replace("/user");
    if (scope === "user" && user.role !== "tenant") router.replace("/admin");
  }, [user, scope, router]);

  if (!user) {
    return (
      <div className="grid min-h-screen place-items-center text-sm text-muted">Loading…</div>
    );
  }

  async function onLogout() {
    await logout();
    router.replace("/login");
  }

  return (
    <div className="grid min-h-screen grid-cols-[220px_1fr]">
      <aside className="sticky top-0 h-screen overflow-y-auto bg-ink/[0.97] text-surface px-2 py-3 flex flex-col gap-1">
        <Link href={scope === "admin" ? "/admin" : "/user"} className="flex items-center gap-2 px-2 py-2 mb-1">
          <div className="grid h-7 w-7 place-items-center rounded-md bg-accent text-accent-fg">
            <Server className="h-3.5 w-3.5" />
          </div>
          <div className="flex flex-col leading-tight">
            <span className="text-[13px] font-semibold tracking-tight">hostQ-cloud</span>
            <span className="text-[10px] text-surface/40 uppercase tracking-widest">{scope}</span>
          </div>
        </Link>

        {groups.map((g, gi) => (
          <div key={gi} className="mt-2">
            {g.title && (
              <div className="px-3 py-1 text-[10px] font-semibold uppercase tracking-widest text-surface/40">
                {g.title}
              </div>
            )}
            <nav className="flex flex-col gap-0.5">
              {g.items.map((item) => {
                const active = item.match ? item.match(pathname) : pathname === item.href || pathname.startsWith(item.href + "/");
                const Icon = item.icon;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={cn(
                      "flex items-center gap-2.5 px-2.5 py-1.5 rounded-md text-[12.5px] font-medium transition-colors",
                      active
                        ? "bg-surface/10 text-surface"
                        : "text-surface/60 hover:bg-surface/5 hover:text-surface"
                    )}
                  >
                    <Icon className="h-3.5 w-3.5" />
                    <span>{item.label}</span>
                  </Link>
                );
              })}
            </nav>
          </div>
        ))}

        <div className="mt-auto pt-3 border-t border-surface/10 px-2 text-[11px] text-surface/40 flex items-center justify-between">
          <span className="truncate">{user.email}</span>
        </div>
      </aside>

      <div className="flex flex-col min-w-0">
        <header className="sticky top-0 z-10 h-12 border-b border-border bg-surface/80 backdrop-blur supports-[backdrop-filter]:bg-surface/60 flex items-center justify-between px-5">
          <div className="flex items-center gap-2 text-sm text-muted">
            <ChevronRight className="h-3.5 w-3.5 opacity-40" />
            <span className="font-medium text-ink">{titleFromPath(pathname)}</span>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" className="gap-2">
              <Search className="h-3.5 w-3.5" />
              <span className="text-xs text-muted">Search</span>
              <kbd className="ml-1 hidden sm:inline-flex h-4 select-none items-center gap-1 rounded border border-border bg-elevated px-1 font-mono text-[10px] text-faint">
                <Command className="h-2.5 w-2.5" />K
              </kbd>
            </Button>
            {mounted && (
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
                title="Toggle theme"
              >
                {theme === "dark" ? <Sun className="h-3.5 w-3.5" /> : <Moon className="h-3.5 w-3.5" />}
              </Button>
            )}
            <Button variant="ghost" size="icon" onClick={onLogout} title="Sign out">
              <LogOut className="h-3.5 w-3.5" />
            </Button>
          </div>
        </header>
        <main className="flex-1 p-5">{children}</main>
      </div>
    </div>
  );
}

function titleFromPath(p: string) {
  const last = p.split("/").filter(Boolean).pop() || "Dashboard";
  return last[0].toUpperCase() + last.slice(1);
}

// Re-exported icon set for nav configs.
export const Icons = {
  LayoutDashboard, Globe, Folder, Database, Shield, Activity,
  Users, ServerCog, Server,
};
