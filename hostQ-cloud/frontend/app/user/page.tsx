"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { apiJSON } from "@/lib/api";
import { PageHead } from "@/components/page-head";
import { Stat } from "@/components/stat";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Globe, Database, Folder, ExternalLink } from "lucide-react";

type Site = { id: string; domain: string; phpVersion: string; suspended: boolean };

export default function UserDashboard() {
  const sites = useQuery({
    queryKey: ["sites"],
    queryFn: () => apiJSON<{ sites: Site[] }>("/api/sites"),
  });
  const list = sites.data?.sites ?? [];

  return (
    <>
      <PageHead title="Your hosting" description="Manage everything on your account." />

      <div className="grid grid-cols-2 lg:grid-cols-3 gap-3 mb-6">
        <Stat label="Sites" icon={Globe} value={list.length} sub={list.length === 1 ? "1 active" : `${list.length} active`} />
        <Stat label="Databases" icon={Database} value="—" />
        <Stat label="Disk used" icon={Folder} value="—" />
      </div>

      <Card>
        <CardContent className="p-0">
          {!list.length ? (
            <div className="p-12 text-center">
              <Globe className="h-7 w-7 mx-auto text-faint mb-3" />
              <div className="text-sm font-medium">No sites yet</div>
              <div className="text-xs text-muted mt-1 mb-4">Create your first site to get started.</div>
              <Link href="/user/sites">
                <Button variant="primary" size="sm">Add a site</Button>
              </Link>
            </div>
          ) : (
            <div className="divide-y divide-border">
              {list.map((s) => (
                <Link
                  key={s.id}
                  href={`/user/sites/${encodeURIComponent(s.domain)}`}
                  className="flex items-center justify-between px-4 py-3 hover:bg-elevated transition-colors"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <div className="grid h-8 w-8 place-items-center rounded-md bg-elevated text-muted shrink-0">
                      <Globe className="h-3.5 w-3.5" />
                    </div>
                    <div className="min-w-0">
                      <div className="text-sm font-medium truncate">{s.domain}</div>
                      <div className="text-xs text-muted">PHP {s.phpVersion}</div>
                    </div>
                  </div>
                  <ExternalLink className="h-3.5 w-3.5 text-faint" />
                </Link>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </>
  );
}
