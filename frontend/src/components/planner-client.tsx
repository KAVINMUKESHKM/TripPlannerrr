"use client";

import dynamic from "next/dynamic";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ApiError, createTrip, currentUser, findAttraction, getAttraction, getRoute, getTrip, searchAttractions, updateTrip } from "@/lib/api";
import type { Attraction, RouteSummary, SearchDay, TripInput, User } from "@/lib/types";
import { AttractionList } from "@/components/attraction-list";
import { ItineraryEditor } from "@/components/itinerary-editor";
import { RouteSummary as RouteSummaryView } from "@/components/route-summary";
import { SiteHeader } from "@/components/site-header";
import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import MenuItem from "@mui/material/MenuItem";
import Tab from "@mui/material/Tab";
import Tabs from "@mui/material/Tabs";
import TextField from "@mui/material/TextField";

const MapView = dynamic(() => import("@/components/maplibre-map"), { ssr: false, loading: () => <div className="map-loading">Preparing your map…</div> });
const PENDING_DRAFT = "tripplanner:pending-trip:v1";
const today = () => { const date = new Date(); date.setMinutes(date.getMinutes() - date.getTimezoneOffset()); return date.toISOString().slice(0, 10); };
const blankDays = (count: number) => Array.from({ length: count }, () => [] as number[]);

type Draft = { destination: string; startDate: string; dayCount: number; selected: number[][]; places: Attraction[] };

export function PlannerClient({ tripId }: { tripId?: string }) {
  const params = useSearchParams();
  const router = useRouter();
  const searchKey = params.toString();
  const [destination, setDestination] = useState(() => params.get("destination") ?? "");
  const [startDate, setStartDate] = useState(() => params.get("date") || today());
  const [dayCount, setDayCount] = useState(() => Math.min(14, Math.max(1, Number(params.get("days") || 3))));
  const [category, setCategory] = useState("");
  const [activeDay, setActiveDay] = useState(1);
  const [suggestions, setSuggestions] = useState<SearchDay[]>([]);
  const [selected, setSelected] = useState<number[][]>(() => blankDays(Math.min(14, Math.max(1, Number(params.get("days") || 3)))));
  const [places, setPlaces] = useState<Attraction[]>([]);
  const [focusPlace, setFocusPlace] = useState<Attraction | null>(null);
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [searchError, setSearchError] = useState("");
  const [saveError, setSaveError] = useState("");
  const [notice, setNotice] = useState("");
  const [route, setRoute] = useState<RouteSummary | null>(null);
  const [routeLoading, setRouteLoading] = useState(false);
  const [routeError, setRouteError] = useState("");
  const [hydrated, setHydrated] = useState(false);
  const firstSearch = useRef(true);

  const allPlaces = useMemo(() => {
    const known = new Map<number, Attraction>();
    for (const place of places) known.set(place.id, place);
    for (const day of suggestions) for (const place of day.attractions) known.set(place.id, place);
    return Array.from(known.values());
  }, [places, suggestions]);
  const selectedIds = useMemo(() => new Set(selected.flat()), [selected]);
  const selectedByDay = useMemo(() => selected.map((ids) => ids.map((id) => findAttraction(suggestions, id) ?? places.find((place) => place.id === id)).filter((place): place is Attraction => Boolean(place))), [places, selected, suggestions]);
  const activePlaces = selectedByDay[activeDay - 1] ?? [];
  const canRoute = activePlaces.length > 1;

  const runSearch = useCallback(async (searchDestination: string, count: number, date: string, selectedCategory: string) => {
    if (!searchDestination.trim()) { setSuggestions([]); setSearchError(""); return; }
    setLoading(true); setSearchError("");
    try {
      const data = await searchAttractions({ destination: searchDestination.trim(), days: count, date, category: selectedCategory || undefined });
      setSuggestions(data);
      setFocusPlace(null);
      setNotice(data.some((day) => day.attractions.length) ? "Places are ready to add to your days." : "No places came back for those dates. Try another destination or category.");
    } catch (cause) {
      setSearchError(cause instanceof ApiError ? cause.message : "We couldn’t reach the planner. Check that the Go API is running and the database is ready.");
    } finally { setLoading(false); }
  }, []);

  useEffect(() => { currentUser().then(setUser).catch(() => setUser(null)); }, []);

  useEffect(() => {
    if (tripId) {
      let live = true;
      setLoading(true);
      getTrip(tripId).then(async (trip) => {
        const ids = trip.days.flatMap((day) => day.stops.map((stop) => stop.attractionId));
        const loaded = await Promise.all(ids.map((id) => getAttraction(id).catch(() => null)));
        if (!live) return;
        const found = loaded.map((place, index): Attraction => place ?? { id: ids[index], name: `Saved stop ${ids[index]}`, latitude: 0, longitude: 0 });
        setPlaces(found);
        const firstDayStop = trip.days.find((day) => day.dayNumber === 1)?.stops[0];
        setFocusPlace(found.find((place) => place.id === firstDayStop?.attractionId) ?? null);
        setDestination(trip.destination);
        setDayCount(trip.dayCount);
        setStartDate(trip.startDate.slice(0, 10));
        setSelected(Array.from({ length: trip.dayCount }, (_, index) => trip.days.find((day) => day.dayNumber === index + 1)?.stops.map((stop) => stop.attractionId) ?? []));
        setActiveDay(1);
        setNotice("Your saved itinerary is open. Changes are saved when you choose Save changes.");
      }).catch((cause) => { if (live) setSearchError(cause instanceof ApiError ? cause.message : "This saved trip could not be loaded."); }).finally(() => { if (live) { setLoading(false); setHydrated(true); } });
      return () => { live = false; };
    }

    if (firstSearch.current) {
      firstSearch.current = false;
      const requestedDestination = params.get("destination") ?? "";
      if (requestedDestination) runSearch(requestedDestination, dayCount, startDate, category);
    }
    setHydrated(true);
  // The URL is the initial search source; later searches are explicit form actions.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tripId, runSearch, searchKey]);

  useEffect(() => {
    if (tripId || !hydrated) return;
    try {
      const raw = window.sessionStorage.getItem(PENDING_DRAFT);
      if (!raw) return;
      const draft = JSON.parse(raw) as Draft;
      if (!draft || typeof draft.destination !== "string" || !Array.isArray(draft.selected) || !Array.isArray(draft.places)) return;
      if (draft.destination !== destination) return;
      setFocusPlace(draft.places.find((place) => place.id === draft.selected[0]?.[0]) ?? null);
      setStartDate(draft.startDate);
      setDayCount(draft.dayCount);
      setSelected(draft.selected);
      setPlaces(draft.places);
      window.sessionStorage.removeItem(PENDING_DRAFT);
      setNotice("Your unsaved plan is back. Save it to keep it with your trips.");
    } catch { window.sessionStorage.removeItem(PENDING_DRAFT); }
  }, [destination, hydrated, tripId]);

  useEffect(() => {
    if (activeDay > dayCount) setActiveDay(dayCount);
  }, [activeDay, dayCount]);

  useEffect(() => {
    if (!canRoute) { setRoute(null); setRouteError(""); return; }
    let live = true;
    setRoute(null); setRouteLoading(true); setRouteError("");
    getRoute(activePlaces.map((place) => ({ latitude: Number(place.latitude), longitude: Number(place.longitude) })))
      .then((result) => { if (live) setRoute(result); })
      .catch(() => { if (live) { setRoute(null); setRouteError("OSRM route details are unavailable right now."); } })
      .finally(() => { if (live) setRouteLoading(false); });
    return () => { live = false; };
  }, [activePlaces, canRoute]);

  function addPlace(place: Attraction) {
    setSelected((current) => current.map((day, index) => index === activeDay - 1 && !day.includes(place.id) ? [...day, place.id] : day));
    setPlaces((current) => current.some((item) => item.id === place.id) ? current : [...current, place]);
    setFocusPlace(null);
    setNotice(`${place.name} added to Day ${activeDay}.`);
  }

  function removePlace(dayNumber: number, id: number) {
    setSelected((current) => current.map((day, index) => index === dayNumber - 1 ? day.filter((placeId) => placeId !== id) : day));
  }

  function movePlace(id: number, fromDay: number, toDay: number) {
    if (fromDay === toDay) return;
    setSelected((current) => current.map((day, index) => {
      if (index === fromDay - 1) return day.filter((placeId) => placeId !== id);
      if (index === toDay - 1 && !day.includes(id)) return [...day, id];
      return day;
    }));
    setActiveDay(toDay);
    setFocusPlace(allPlaces.find((place) => place.id === id) ?? null);
  }

  function reorder(dayNumber: number, index: number, direction: -1 | 1) {
    setSelected((current) => current.map((day, dayIndex) => {
      if (dayIndex !== dayNumber - 1) return day;
      const target = index + direction;
      if (target < 0 || target >= day.length) return day;
      const next = [...day]; [next[index], next[target]] = [next[target], next[index]]; return next;
    }));
  }

  async function saveTrip() {
    setSaveError("");
    if (!destination.trim()) { setSaveError("Add a destination before saving this trip."); return; }
    if (!user) {
      try { window.sessionStorage.setItem(PENDING_DRAFT, JSON.stringify({ destination, startDate, dayCount, selected, places } satisfies Draft)); } catch { /* The current plan remains editable in this tab. */ }
      const next = tripId ? `/trips/${tripId}` : `/plan?${new URLSearchParams({ destination, date: startDate, days: String(dayCount) })}`;
      router.push(`/login?next=${encodeURIComponent(next)}`);
      return;
    }
    setSaving(true);
    const input: TripInput = {
      destination: destination.trim(),
      startDate: new Date(`${startDate}T00:00:00.000Z`).toISOString(),
      dayCount,
      days: selected.map((attractionIds, index) => ({ dayNumber: index + 1, attractionIds })),
    };
    try {
      const saved = tripId ? await updateTrip(tripId, input) : await createTrip(input);
      window.sessionStorage.removeItem(PENDING_DRAFT);
      router.push(`/trips/${saved.id}`);
    } catch (cause) {
      setSaveError(cause instanceof ApiError ? cause.message : "Your trip couldn’t be saved. Check your connection and try again.");
    } finally { setSaving(false); }
  }

  function submitSearch(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setNotice("");
    const nextCount = Math.min(14, Math.max(1, dayCount));
    setDayCount(nextCount);
    setSelected((current) => Array.from({ length: nextCount }, (_, index) => current[index] ?? []));
    setActiveDay(1);
    runSearch(destination, nextCount, startDate, category);
    const query = new URLSearchParams({ destination, days: String(nextCount), date: startDate });
    window.history.replaceState(null, "", `/plan?${query.toString()}`);
  }

  return <main className="planner-shell">
    <SiteHeader />
    <section className="planner-heading"><div><Link className="back-link" href={tripId ? "/trips" : "/"}>← {tripId ? "Saved trips" : "Home"}</Link><span className="eyebrow">A trip takes shape one stop at a time</span><h1>{tripId ? "Edit your itinerary." : "Make a day of it."}</h1><p>Gather a few places, give them an order, and leave some space between.</p></div>
      <Button className="save-button" variant="contained" onClick={saveTrip} disabled={saving}>{saving ? "Saving…" : tripId ? "Save changes" : user ? "Save this trip" : "Sign in to save"}</Button>
    </section>

    <form className="planner-search" onSubmit={submitSearch}>
      <div className="search-field"><label htmlFor="planner-destination">Destination</label><TextField id="planner-destination" value={destination} onChange={(event) => setDestination(event.target.value)} placeholder="City or place" required /></div>
      <div className="search-field"><label htmlFor="planner-date">First day</label><TextField id="planner-date" type="date" value={startDate} onChange={(event) => setStartDate(event.target.value)} required /></div>
      <div className="search-field"><label htmlFor="planner-days">Days</label><TextField id="planner-days" select value={dayCount} onChange={(event) => { const count = Number(event.target.value); setDayCount(count); setSelected((current) => Array.from({ length: count }, (_, index) => current[index] ?? [])); }} >{Array.from({ length: 14 }, (_, index) => <MenuItem key={index} value={index + 1}>{index + 1} {index === 0 ? "day" : "days"}</MenuItem>)}</TextField></div>
      <div className="search-field"><label htmlFor="planner-interests">Interests</label><TextField id="planner-interests" select value={category} onChange={(event) => setCategory(event.target.value)}><MenuItem value="">Any kind</MenuItem><MenuItem value="Nature">Nature</MenuItem><MenuItem value="History">History</MenuItem><MenuItem value="Culture">Culture</MenuItem><MenuItem value="Food">Food</MenuItem></TextField></div>
      <Button variant="contained" type="submit" disabled={loading}>{loading ? "Finding places…" : "Find places"}</Button>
    </form>

    {searchError ? <Alert className="notice" severity="error" action={<Button color="inherit" size="small" onClick={() => runSearch(destination, dayCount, startDate, category)}>Try again</Button>}><strong>Places aren’t available yet.</strong> {searchError} Start the Go API and make sure its database migrations are applied, then search again.</Alert> : null}
    {saveError ? <Alert className="notice" severity="error" role="alert">{saveError}</Alert> : null}
    {notice ? <p className="notice notice-good" role="status">{notice}</p> : null}

    <div className="planner-grid">
      <div className="planner-main">
        <Tabs className="day-tabs" value={activeDay} onChange={(_, value: number) => { setActiveDay(value); setFocusPlace(selectedByDay[value - 1]?.[0] ?? null); }} variant="scrollable" scrollButtons="auto" aria-label="Choose a day to plan">{Array.from({ length: dayCount }, (_, index) => <Tab className="day-tab" key={index} value={index + 1} label={<><span>Day {index + 1}</span><small>{selected[index]?.length ?? 0} stops</small></>} />)}</Tabs>
        <AttractionList attractions={allPlaces} selectedIds={selectedIds} onAdd={addPlace} onFocus={setFocusPlace} />
      </div>
      <aside className="planner-side" aria-label="Current trip details">
        <div className="map-heading"><div><span className="eyebrow">Your route at a glance</span><h2>{destination || "Your destination"}</h2></div><span className="map-provider">OpenFreeMap</span></div>
        <MapView places={activePlaces} route={route} focusPlace={focusPlace} />
        <RouteSummaryView route={route} loading={routeLoading} error={routeError} stopCount={activePlaces.length} />
        <ItineraryEditor days={selectedByDay} suggestions={suggestions} onRemove={removePlace} onMove={movePlace} onReorder={reorder} />
        {!user ? <div className="save-prompt"><strong>Your plan stays in this tab.</strong><p>Sign in when you’re ready to save it and pick it up later.</p><Link className="quiet-link" href={`/login?next=${encodeURIComponent(`/plan?${new URLSearchParams({ destination, date: startDate, days: String(dayCount) })}`)}`}>Sign in or create an account ↗</Link></div> : null}
      </aside>
    </div>
  </main>;
}
