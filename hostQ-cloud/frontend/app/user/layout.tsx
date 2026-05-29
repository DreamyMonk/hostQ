"use client";

import { Shell, type NavGroup, Icons } from "@/components/shell";

const groups: NavGroup[] = [
  {
    title: "Overview",
    items: [
      { href: "/user", label: "Dashboard", icon: Icons.LayoutDashboard },
    ],
  },
  {
    title: "Hosting",
    items: [
      { href: "/user/sites", label: "My Sites", icon: Icons.Globe },
      { href: "/user/files", label: "Files", icon: Icons.Folder },
      { href: "/user/databases", label: "Databases", icon: Icons.Database },
    ],
  },
];

export default function UserLayout({ children }: { children: React.ReactNode }) {
  return <Shell scope="user" groups={groups}>{children}</Shell>;
}
