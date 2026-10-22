"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { ApiError, deleteTrip, listTrips } from "@/lib/api";
import type { Trip } from "@/lib/types";
import { SiteHeader } from "@/components/site-header";

export default function TripsPage() {
  const [trips, setTrips] = useState<Trip[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [deleting, setDeleting] = useState<number | null>(null);
  const load = useCallback(async () => {
    setLoading(true); setError("");
    try { setTrips(await listTrips()); }
    catch (cause) { setError(cause instanceof ApiError && cause.status === 401 ? "Sign in to see trips you’ve saved." : cause instanceof ApiError ? cause.message : "The trip list couldn’t be reached. Check that the API is running."); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { load(); }, [load]);

  async function removeTrip(trip: Trip) {
    if (!window.confirm(`Delete your trip to ${trip.destination}? This can’t be undone.`)) return;
    setDeleting(trip.id); setError("");
    try { await deleteTrip(trip.id); setTrips((current) => current.filter((item) => item.id !== trip.id)); }
    catch (cause) { setError(cause instanceof ApiError ? cause.message : "That trip could not be deleted. Try again."); }
    finally { setDeleting(null); }
  }

  return <main className="trips-shell"><SiteHeader /><section className="trips-heading"><div><Link className="back-link" href="/">← Home</Link><span className="eyebrow">Your travel notebook</span><h1>Trips worth keeping.</h1><p>Every saved plan is here when you’re ready to pick it up again.</p></div><Link className="button" href="/plan">Plan a trip <span aria-hidden="true">↗</span></Link></section>
    {error ? <div className="notice notice-error" role="alert"><span>{error}</span>{error.includes("Sign in") ? <Link href="/login?next=%2Ftrips">Sign in</Link> : <button className="text-button" onClick={load}>Try again</button>}</div> : null}
    {loading ? <div className="loading-line" role="status">Gathering your saved trips…</div> : trips.length ? <div className="trip-grid">{trips.map((trip) => <article className="trip-card" key={trip.id}><Link href={`/trips/${trip.id}`} className="trip-card-link"><div className="trip-card-photo"><span className="trip-card-mark">{trip.destination.slice(0, 1).toUpperCase()}</span></div><div className="trip-card-copy"><span className="eyebrow">{new Date(`${trip.startDate.slice(0, 10)}T12:00:00`).toLocaleDateString(undefined, { month: "long", year: "numeric" })}</span><h2>{trip.destination}</h2><p>{trip.dayCount} {trip.dayCount === 1 ? "day" : "days"} · {trip.days.reduce((total, day) => total + day.stops.length, 0)} stops planned</p><span className="quiet-link">Open itinerary ↗</span></div></Link><button className="trip-delete" disabled={deleting === trip.id} onClick={() => removeTrip(trip)} aria-label={`Delete trip to ${trip.destination}`}>{deleting === trip.id ? "Deleting…" : "Delete trip"}</button></article>)}</div> : !error && !loading ? <section className="trips-empty"><span className="empty-compass" aria-hidden="true">✳</span><span className="eyebrow">Your notebook is open</span><h2>No saved trips yet.</h2><p>Start with a destination and keep the places you find along the way.</p><Link className="button" href="/plan">Plan your first trip</Link></section> : null}
  </main>;
}
