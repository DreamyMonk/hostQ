import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "hostQ - Hosting Control Panel",
  description: "Self-hosted hosting control panel. Manage sites, PHP, databases, SSL, WordPress installs, and files from one dashboard.",
  keywords: "hosting panel, control panel, PHP manager, SSL, WordPress installer, database manager",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head>
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" href="/favicon.ico" />
      </head>
      <body>{children}</body>
    </html>
  );
}
