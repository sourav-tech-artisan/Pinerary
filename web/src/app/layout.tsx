import type { Metadata, Viewport } from "next";
import "@fontsource-variable/dm-sans/wght.css";
import "@fontsource-variable/fraunces/wght.css";
import "maplibre-gl/dist/maplibre-gl.css";
import "./globals.css";
import { AppShell } from "@/components/app-shell";
import { AppProvider } from "@/components/app-provider";

export const metadata: Metadata = {
  title: {
    default: "Pinerary",
    template: "%s · Pinerary",
  },
  description: "Keep the route, places, and stories from every trip.",
  applicationName: "Pinerary",
  manifest: "/manifest.webmanifest",
  appleWebApp: {
    capable: true,
    statusBarStyle: "black-translucent",
    title: "Pinerary",
  },
  formatDetection: {
    telephone: false,
  },
};

export const viewport: Viewport = {
  themeColor: "#123f35",
  colorScheme: "light",
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en">
      <body>
        <AppProvider>
          <AppShell>{children}</AppShell>
        </AppProvider>
      </body>
    </html>
  );
}
