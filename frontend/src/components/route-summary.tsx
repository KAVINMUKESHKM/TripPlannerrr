import type { RouteSummary as RouteData } from "@/lib/types";

export function RouteSummary({ route, loading, error, stopCount }: { route: RouteData | null; loading: boolean; error: string; stopCount: number }) {
  if (stopCount < 2) return <div className="route-summary route-empty"><span className="route-symbol" aria-hidden="true">⌁</span><p>Add another stop to see how the route comes together.</p></div>;
  if (loading) return <div className="route-summary"><span className="route-symbol" aria-hidden="true">⌁</span><p>Finding a route between your stops…</p></div>;
  if (error || !route) return <div className="route-summary route-warning"><span className="route-symbol" aria-hidden="true">⌁</span><p>{error || "Route details are unavailable. Your stops are still saved."}</p></div>;
  const distance = route.distanceMeters >= 1000 ? `${(route.distanceMeters / 1000).toFixed(1)} km` : `${Math.round(route.distanceMeters)} m`;
  const minutes = Math.max(1, Math.round(route.durationSeconds / 60));
  return <div className="route-summary"><span className="route-symbol" aria-hidden="true">⌁</span><div><span className="eyebrow">Between these stops</span><p><strong>{distance}</strong><span className="route-separator">·</span>{minutes < 60 ? `${minutes} min drive` : `${Math.floor(minutes / 60)} hr ${minutes % 60} min drive`}</p></div></div>;
}
