"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiJSON } from "@/lib/api";
import { PageHead } from "@/components/page-head";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogTrigger } from "@/components/ui/dialog";
import { Plus, Globe, Trash2 } from "lucide-react";

type Site = { id: string; domain: string; ownerId: string; phpVersion: string; suspended: boolean; createdAt: string };

export default function AdminSites() {
  const qc = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ["sites"],
    queryFn: () => apiJSON<{ sites: Site[] }>("/api/sites"),
  });

  const [open, setOpen] = useState(false);
  const [domain, setDomain] = useState("");
  const [phpVersion, setPhpVersion] = useState("8.3");
  const [err, setErr] = useState<string | null>(null);

  const create = useMutation({
    mutationFn: (body: { domain: string; phpVersion: string }) =>
      apiJSON<{ id: string; domain: string }>("/api/sites", { method: "POST", json: body }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["sites"] });
      setOpen(false);
      setDomain("");
      setErr(null);
    },
    onError: (e: any) => setErr(e?.message || "Failed to create site"),
  });

  const del = useMutation({
    mutationFn: (id: string) => apiJSON(`/api/sites/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["sites"] }),
  });

  return (
    <>
      <PageHead
        title="All sites"
        description="Every site across every tenant on this server."
        actions={
          <Dialog open={open} onOpenChange={setOpen}>
            <DialogTrigger asChild>
              <Button variant="primary">
                <Plus className="h-3.5 w-3.5" />
                New site
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create a site</DialogTitle>
                <DialogDescription>nginx + apache vhosts get written immediately.</DialogDescription>
              </DialogHeader>
              <form
                className="space-y-3"
                onSubmit={(e) => {
                  e.preventDefault();
                  create.mutate({ domain, phpVersion });
                }}
              >
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-muted">Domain</label>
                  <Input placeholder="example.com" value={domain} onChange={(e) => setDomain(e.target.value)} required />
                </div>
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-muted">PHP version</label>
                  <select
                    className="flex h-9 w-full rounded-md border border-border bg-surface px-3 text-sm"
                    value={phpVersion}
                    onChange={(e) => setPhpVersion(e.target.value)}
                  >
                    <option value="8.3">8.3</option>
                    <option value="8.2">8.2</option>
                    <option value="8.1">8.1</option>
                  </select>
                </div>
                {err && <div className="rounded-md border border-danger/30 bg-danger/5 px-3 py-2 text-xs text-danger">{err}</div>}
                <DialogFooter>
                  <Button type="button" variant="ghost" onClick={() => setOpen(false)}>Cancel</Button>
                  <Button type="submit" variant="primary" disabled={create.isPending}>
                    {create.isPending ? "Creating…" : "Create site"}
                  </Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
        }
      />

      <Card>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="p-8 text-center text-xs text-muted">Loading sites…</div>
          ) : !data?.sites?.length ? (
            <div className="p-12 text-center">
              <Globe className="h-7 w-7 mx-auto text-faint mb-3" />
              <div className="text-sm font-medium">No sites yet</div>
              <div className="text-xs text-muted mt-1">Create the first site to get started.</div>
            </div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-[10.5px] font-semibold uppercase tracking-wider text-muted border-b border-border">
                  <th className="text-left px-4 py-2.5">Domain</th>
                  <th className="text-left px-4 py-2.5">PHP</th>
                  <th className="text-left px-4 py-2.5">Status</th>
                  <th className="text-left px-4 py-2.5">Created</th>
                  <th className="text-right px-4 py-2.5"></th>
                </tr>
              </thead>
              <tbody>
                {data.sites.map((s) => (
                  <tr key={s.id} className="border-b border-border last:border-0 hover:bg-elevated">
                    <td className="px-4 py-2.5 font-medium">{s.domain}</td>
                    <td className="px-4 py-2.5 text-muted">{s.phpVersion}</td>
                    <td className="px-4 py-2.5">
                      {s.suspended ? (
                        <span className="inline-flex items-center rounded border border-warning/30 bg-warning/10 px-1.5 py-0.5 text-[10.5px] font-semibold text-warning">Suspended</span>
                      ) : (
                        <span className="inline-flex items-center rounded border border-success/30 bg-success/10 px-1.5 py-0.5 text-[10.5px] font-semibold text-success">Active</span>
                      )}
                    </td>
                    <td className="px-4 py-2.5 text-muted text-xs">{new Date(s.createdAt).toLocaleDateString()}</td>
                    <td className="px-4 py-2.5 text-right">
                      <Button
                        variant="danger"
                        size="icon"
                        onClick={() => confirm(`Delete ${s.domain}? Vhosts will be removed.`) && del.mutate(s.id)}
                        title="Delete site"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>
    </>
  );
}
