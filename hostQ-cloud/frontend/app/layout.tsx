import type { Metadata } from "next";
import "./globals.css";
import { Providers } from "./providers";

export const metadata: Metadata = {
  title: "hostQ-cloud",
  description: "Premium multi-tenant hosting control panel.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className="min-h-screen bg-canvas font-sans antialiased text-ink">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
