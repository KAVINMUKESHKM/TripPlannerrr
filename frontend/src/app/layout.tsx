import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = { title: { default: "Trip Planner — Make a trip of your own", template: "%s · Trip Planner" }, description: "Find places worth stopping for, shape your days, and keep your itinerary close." };
export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="en"><body>{children}</body></html>;
}
