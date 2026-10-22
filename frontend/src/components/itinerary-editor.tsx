"use client";

import type { Attraction, SearchDay } from "@/lib/types";

export function ItineraryEditor({ days, suggestions, onRemove, onMove, onReorder }: {
  days: Attraction[][];
  suggestions: SearchDay[];
  onRemove: (dayNumber: number, attractionId: number) => void;
  onMove: (attractionId: number, fromDay: number, toDay: number) => void;
  onReorder: (dayNumber: number, index: number, direction: -1 | 1) => void;
}) {
  return <section className="itinerary-panel" aria-labelledby="itinerary-title">
    <div className="panel-heading"><div><span className="eyebrow">Your days, taking shape</span><h2 id="itinerary-title">The itinerary</h2></div><span className="result-count">{days.reduce((total, day) => total + day.length, 0)} stops</span></div>
    <div className="day-stack">{days.map((places, dayIndex) => {
      const number = dayIndex + 1;
      const forecast = suggestions.find((day) => day.day === number);
      return <section className="day-card" key={number} aria-labelledby={`day-${number}`}>
        <div className="day-heading"><div className="day-number">{String(number).padStart(2, "0")}</div><div><h3 id={`day-${number}`}>Day {number}</h3><p>{forecast?.date ? new Date(`${forecast.date}T12:00:00`).toLocaleDateString(undefined, { weekday: "long", month: "short", day: "numeric" }) : number === 1 ? "Start anywhere" : "Leave space to wander"}{forecast?.weather ? ` · ${forecast.weather}${forecast.temperature == null ? "" : `, ${Math.round(forecast.temperature)}°`}` : ""}</p></div></div>
        {places.length ? <ol className="stop-list">{places.map((place, index) => <li className="stop-row" key={place.id}>
          <span className="stop-index">{index + 1}</span><div className="stop-copy"><strong>{place.name}</strong><span>{place.category || "A place to explore"}</span></div>
          <div className="stop-actions">
            <button className="icon-button" onClick={() => onReorder(number, index, -1)} disabled={index === 0} aria-label={`Move ${place.name} earlier`}>↑</button>
            <button className="icon-button" onClick={() => onReorder(number, index, 1)} disabled={index === places.length - 1} aria-label={`Move ${place.name} later`}>↓</button>
            <label className="sr-only" htmlFor={`move-${number}-${place.id}`}>Move {place.name} to another day</label><select id={`move-${number}-${place.id}`} value={number} onChange={(event) => onMove(place.id, number, Number(event.target.value))} aria-label={`Move ${place.name} to another day`}>{days.map((_, index) => <option key={index + 1} value={index + 1}>Day {index + 1}</option>)}</select>
            <button className="icon-button remove-stop" onClick={() => onRemove(number, place.id)} aria-label={`Remove ${place.name}`}>×</button>
          </div>
        </li>)}</ol> : <p className="day-empty">No stops yet. Pick a place from the list to add one.</p>}
      </section>;
    })}</div>
  </section>;
}
