"use client";

import { Shell, type NavGroup, Icons } from "@/components/shell";

const groups: NavGroup[] = [
  {
    title: "Overview",
    items: [
      { href: "/admin", label: "Dashboard", icon: Icons.LayoutDashboard },
    ],
  },
  {
    title: "Manage",
    items: [
      { href: "/admin/users", label: "Users & Tenants", icon: Icons.Users },
      { href: "/admin/sites", label: "All Sites", icon: Icons.Globe },
      { href: "/admin/databases", label: "Databases", icon: Icons.Database },
    ],
  },
  {
    title: "Server",
    items: [
      { href: "/admin/services", label: "Services", icon: Icons.ServerCog },
      { href: "/admin/firewall", label: "Firewall", icon: Icons.Shield },
      { href: "/admin/audit", label: "Audit log", icon: Icons.Activity },
    ],
  },
];

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return <Shell scope="admin" groups={groups}>{children}</Shell>;
}
