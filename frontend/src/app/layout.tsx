import type { Metadata } from "next";
import { AppRouterCacheProvider } from "@mui/material-nextjs/v16-appRouter";
import { AppProviders } from "@/components/app-providers";
import { headers } from "next/headers";
import "./globals.css";

export const metadata: Metadata = { title: { default: "Trip Planner — Make a trip of your own", template: "%s · Trip Planner" }, description: "Find places worth stopping for, shape your days, and keep your itinerary close." };
export default async function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  const nonce = (await headers()).get("x-nonce") ?? undefined;
  return <html lang="en"><body><AppRouterCacheProvider options={{ enableCssLayer: true, nonce }}><AppProviders>{children}</AppProviders></AppRouterCacheProvider></body></html>;
}
