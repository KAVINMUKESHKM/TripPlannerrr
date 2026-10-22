"use client";

import Image from "next/image";
import { useMemo, useState } from "react";
import type { Attraction } from "@/lib/types";

export function AttractionList({ attractions, selectedIds, onAdd }: { attractions: Attraction[]; selectedIds: Set<number>; onAdd: (place: Attraction) => void }) {
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("All places");
  const categories = useMemo(() => ["All places", ...Array.from(new Set(attractions.map((place) => place.category || "Place"))).sort()], [attractions]);
  const filtered = useMemo(() => attractions.filter((place) => {
    const matchesText = `${place.name} ${place.description ?? ""} ${place.category ?? ""}`.toLowerCase().includes(query.toLowerCase());
    return matchesText && (category === "All places" || place.category === category);
  }), [attractions, category, query]);

  return <section className="places-panel" aria-labelledby="places-title">
    <div className="panel-heading"><div><span className="eyebrow">Places to consider</span><h2 id="places-title">Make a few good stops.</h2></div><span className="result-count">{filtered.length} places</span></div>
    <label className="place-search"><span className="sr-only">Search places</span><span aria-hidden="true">⌕</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search by name or kind" /></label>
    <div className="category-row" aria-label="Filter by category">{categories.map((item) => <button className={`category-chip${category === item ? " is-active" : ""}`} key={item} onClick={() => setCategory(item)} aria-pressed={category === item}>{item}</button>)}</div>
    {filtered.length ? <ul className="place-list">{filtered.map((place) => {
      const added = selectedIds.has(place.id);
      return <li className="place-card" key={place.id}>
        <div className="place-image">{place.image ? <Image src={place.image} alt="" fill sizes="120px" /> : <span aria-hidden="true">{place.category?.toLowerCase().includes("museum") ? "⌂" : place.category?.toLowerCase().includes("nature") ? "⌁" : "✳"}</span>}</div>
        <div className="place-info"><span className="place-kind">{place.category || "Worth a stop"}</span><h3>{place.name}</h3><p>{place.description || "A place to take your time and have a look around."}</p></div>
        <button className={`add-place${added ? " added" : ""}`} onClick={() => onAdd(place)} disabled={added} aria-label={added ? `${place.name} already added` : `Add ${place.name} to your day`}>{added ? "Added" : "Add"}</button>
      </li>;
    })}</ul> : <div className="empty-inline"><strong>No places match that search.</strong><span>Try another name or clear the category filter.</span></div>}
  </section>;
}
