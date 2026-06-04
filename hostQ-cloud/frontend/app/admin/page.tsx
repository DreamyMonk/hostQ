"use client";

import { useQuery } from "@tanstack/react-query";
import { apiJSON } from "@/lib/api";
import { PageHead } from "@/components/page-head";
import { Stat } from "@/components/stat";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Users, Globe, Activity, Database } from "lucide-react";

type Site = { id: string; domain: string; ownerId: string; createdAt: string; suspended: boolean };

export default function AdminDashboard() {
  const sites = useQuery({
    queryKey: ["sites"],
    queryFn: () => apiJSON<{ sites: Site[] }>("/api/sites"),
  });

  const totalSites = sites.data?.sites?.length ?? 0;
  const suspended = sites.data?.sites?.filter((s) => s.suspended).length ?? 0;

  return (
    <>
      <PageHead title="Server overview" description="Everything across all tenants — at a glance." />

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 mb-6">
        <Stat label="Sites" icon={Globe} value={totalSites} sub={suspended ? `${suspended} suspended` : "All running"} />
        <Stat label="Tenants" icon={Users} value="—" sub="Coming soon" />
        <Stat label="Databases" icon={Database} value="—" sub="Coming soon" />
        <Stat label="Today" icon={Activity} value="—" sub="Coming soon" />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Recent activity</CardTitle>
          <CardDescription>The last actions taken on this server.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-xs text-muted py-8 text-center">
            Audit feed will appear here once the audit-log endpoint is wired up.
          </div>
        </CardContent>
      </Card>
    </>
  );
}
