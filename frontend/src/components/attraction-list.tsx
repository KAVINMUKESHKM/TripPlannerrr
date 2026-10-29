"use client";

import { useMemo, useState } from "react";
import type { Attraction, NearbyRestaurant } from "@/lib/types";
import { getNearbyRestaurants, ApiError } from "@/lib/api";
import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Collapse from "@mui/material/Collapse";
import CircularProgress from "@mui/material/CircularProgress";
import TextField from "@mui/material/TextField";

function externalURL(value: string): string | undefined {
  try {
    const normalized = /^[a-z][a-z0-9+.-]*:/i.test(value) ? value : `https://${value}`;
    const url = new URL(normalized);
    return url.protocol === "https:" || url.protocol === "http:" ? url.href : undefined;
  } catch {
    return undefined;
  }
}

function wikipediaURL(value: string): string | undefined {
  const [language, ...title] = value.split(":");
  if (!/^[a-z][a-z0-9-]{1,11}$/i.test(language) || title.length === 0) return undefined;
  return `https://${language.toLowerCase()}.wikipedia.org/wiki/${encodeURIComponent(title.join(":").trim().replaceAll(" ", "_"))}`;
}

function wheelchairLabel(value: string): string {
  const labels: Record<string, string> = { yes: "Wheelchair access · Yes", no: "Wheelchair access · No", limited: "Wheelchair access · Limited", designated: "Wheelchair access · Designated" };
  return labels[value.toLowerCase()] ?? `Wheelchair access · ${value}`;
}

export function AttractionList({ attractions, selectedIds, onAdd, onFocus }: { attractions: Attraction[]; selectedIds: Set<number>; onAdd: (place: Attraction) => void; onFocus: (place: Attraction) => void }) {
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("All places");
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [restaurantResults, setRestaurantResults] = useState<Record<number, { loading: boolean; error: string; items: NearbyRestaurant[] }>>({});
  const categories = useMemo(() => ["All places", ...Array.from(new Set(attractions.map((place) => place.category || "Place"))).sort()], [attractions]);
  const filtered = useMemo(() => attractions.filter((place) => {
    const matchesText = `${place.name} ${place.description ?? ""} ${place.category ?? ""}`.toLowerCase().includes(query.toLowerCase());
    return matchesText && (category === "All places" || place.category === category);
  }), [attractions, category, query]);

  async function toggleRestaurants(id: number) {
    if (expandedId === id) { setExpandedId(null); return; }
    setExpandedId(id);
    if (restaurantResults[id]?.items.length || restaurantResults[id]?.loading) return;
    setRestaurantResults((current) => ({ ...current, [id]: { loading: true, error: "", items: [] } }));
    try {
      const items = await getNearbyRestaurants(id);
      setRestaurantResults((current) => ({ ...current, [id]: { loading: false, error: "", items } }));
    } catch (cause) {
      const error = cause instanceof ApiError && cause.status === 503
        ? "Nearby restaurant search is not configured. Add SERP_API_KEY to backend/.env."
        : cause instanceof Error ? cause.message : "Nearby restaurants could not be loaded.";
      setRestaurantResults((current) => ({ ...current, [id]: { loading: false, error, items: [] } }));
    }
  }

  return <section className="places-panel" aria-labelledby="places-title">
    <div className="panel-heading"><div><span className="eyebrow">Places to consider</span><h2 id="places-title">Make a few good stops.</h2></div><span className="result-count">{filtered.length} places</span></div>
    <div className="search-field place-search-field"><label htmlFor="place-search">Search places</label><TextField id="place-search" className="place-search-input" fullWidth size="small" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search by name or kind" /></div>
    <div className="category-row" aria-label="Filter by category">{categories.map((item) => <Chip className="category-chip" key={item} label={item} onClick={() => setCategory(item)} color={category === item ? "primary" : "default"} variant={category === item ? "filled" : "outlined"} />)}</div>
    <div className="places-results-scroll" role="region" aria-label="Suggested places" tabIndex={0}>
    {filtered.length ? <ul className="place-list">{filtered.map((place) => {
      const added = selectedIds.has(place.id);
      const website = place.website ? externalURL(place.website) : undefined;
      const wikipedia = place.wikipedia ? wikipediaURL(place.wikipedia) : undefined;
      const wikidata = place.wikidata && /^Q[1-9]\d*$/.test(place.wikidata) ? `https://www.wikidata.org/wiki/${place.wikidata}` : undefined;
      const hasVisitorDetails = place.address || place.opening_hours || place.wheelchair || place.fee || place.toilets || place.toilets_wheelchair || place.internet_access || website || place.phone || wikipedia || wikidata;
      return <li className="place-card" key={place.id}>
        <div className="place-info"><span className="place-kind">{place.category || "Worth a stop"}</span><h3><button className="place-focus" type="button" onClick={() => onFocus(place)} aria-label={`Show ${place.name} on the map`}>{place.name}</button></h3><p>{place.description || "No description is available from the mapped sources yet."}</p>{place.source_url ? <a className="place-source" href={place.source_url} target="_blank" rel="noreferrer">OpenStreetMap ↗</a> : null}</div>
        <Button className="add-place" size="small" variant={added ? "outlined" : "contained"} onClick={() => onAdd(place)} disabled={added} aria-label={added ? `${place.name} already added` : `Add ${place.name} to your day`}>{added ? "Added" : "Add"}</Button>
        <div className="place-actions-extra"><Button size="small" onClick={() => toggleRestaurants(place.id)} aria-expanded={expandedId === place.id}>{expandedId === place.id ? "Hide details" : "Details & nearby food"}</Button></div>
        <Collapse className="restaurant-collapse" in={expandedId === place.id} unmountOnExit>
        <div className="restaurant-panel">
          <h4>Visiting details</h4>
          {hasVisitorDetails ? <div className="attraction-detail-facts">
            {place.address ? <span><strong>Address</strong>{place.address}</span> : null}
            {place.opening_hours ? <span><strong>Hours</strong>{place.opening_hours}</span> : null}
            {place.wheelchair ? <span><strong>Accessibility</strong>{wheelchairLabel(place.wheelchair)}</span> : null}
            {place.fee ? <span><strong>Entry fee</strong>{place.fee === "yes" ? "Fee may apply" : place.fee === "no" ? "No fee listed" : place.fee}</span> : null}
            {place.toilets ? <span><strong>Toilets</strong>{place.toilets === "yes" ? "Available" : place.toilets}</span> : null}
            {place.toilets_wheelchair ? <span><strong>Accessible toilets</strong>{place.toilets_wheelchair}</span> : null}
            {place.internet_access ? <span><strong>Internet access</strong>{place.internet_access}</span> : null}
            {place.phone ? <span><strong>Phone</strong>{place.phone}</span> : null}
            {website ? <a href={website} target="_blank" rel="noopener noreferrer">Official website ↗</a> : null}
            {wikipedia ? <a href={wikipedia} target="_blank" rel="noopener noreferrer">Wikipedia ↗</a> : null}
            {wikidata ? <a href={wikidata} target="_blank" rel="noopener noreferrer">Wikidata ↗</a> : null}
          </div> : <p className="restaurant-empty">No visiting details have been mapped for this place yet.</p>}
          {place.details_updated_at ? <p className="restaurant-empty">OpenStreetMap details checked {new Date(place.details_updated_at).toLocaleDateString()} · confirm hours before visiting.</p> : null}
          <div className="attraction-detail-facts">
            {place.best_climate ? <span><strong>Best climate</strong>{place.best_climate}</span> : null}
            {place.ideal_weather ? <span><strong>Ideal weather</strong>{place.ideal_weather}</span> : null}
            {place.ideal_temp_min != null || place.ideal_temp_max != null ? <span><strong>Ideal temperature</strong>{place.ideal_temp_min ?? "—"}°C to {place.ideal_temp_max ?? "—"}°C</span> : null}
            <a href={`https://www.google.com/maps/search/?api=1&query=${place.latitude},${place.longitude}`} target="_blank" rel="noreferrer">View location on Google Maps ↗</a>
          </div>
          <h4>Nearby restaurants</h4>
          {restaurantResults[place.id]?.loading ? <div className="restaurant-loading"><CircularProgress size={18} /><span>Finding nearby restaurants…</span></div> : null}
          {restaurantResults[place.id]?.error ? <Alert severity="info">{restaurantResults[place.id].error}</Alert> : null}
          {!restaurantResults[place.id]?.loading && !restaurantResults[place.id]?.error && restaurantResults[place.id]?.items.length === 0 ? <p className="restaurant-empty">No nearby restaurants were found.</p> : null}
          {restaurantResults[place.id]?.items.map((restaurant) => <article className="restaurant-result" key={`${restaurant.name}-${restaurant.latitude}-${restaurant.longitude}`}>
            <div><strong>{restaurant.name}</strong><span>{restaurant.address || "Address unavailable"}</span></div>
            <span className="restaurant-rating">★ {restaurant.rating.toFixed(1)} · {restaurant.distanceKm.toFixed(1)} km</span>
            <a href={`https://www.google.com/maps/search/?api=1&query=${restaurant.latitude},${restaurant.longitude}`} target="_blank" rel="noreferrer">Map ↗</a>
          </article>)}
          {restaurantResults[place.id]?.error ? <Button size="small" onClick={() => { setRestaurantResults((current) => { const next = { ...current }; delete next[place.id]; return next; }); void toggleRestaurants(place.id); }}>Try again</Button> : null}
        </div>
        </Collapse>
      </li>;
    })}</ul> : <div className="empty-inline"><strong>No places match that search.</strong><span>Try another name or clear the category filter.</span></div>}
    </div>
    {attractions.some((place) => place.source_url) ? <p className="data-attribution">Place data © <a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noreferrer">OpenStreetMap contributors</a>, available under the ODbL.</p> : null}
  </section>;
}
