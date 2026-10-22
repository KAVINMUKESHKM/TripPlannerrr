"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

export function TripSearchForm() {
  const router = useRouter();
  const [destination, setDestination] = useState("");
  const [startDate, setStartDate] = useState(() => { const date = new Date(); date.setMinutes(date.getMinutes() - date.getTimezoneOffset()); return date.toISOString().slice(0, 10); });
  const [days, setDays] = useState("3");
  const [message, setMessage] = useState("");

  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!destination.trim()) { setMessage("Enter a city or place to get started."); return; }
    const params = new URLSearchParams({ destination: destination.trim(), days, date: startDate });
    router.push(`/plan?${params.toString()}`);
  }

  return (
    <form className="search-rail" onSubmit={submit}>
      <label className="search-field search-destination"><span>Where to?</span><input value={destination} onChange={(event) => { setDestination(event.target.value); setMessage(""); }} placeholder="A city, coast, or mountain town" autoComplete="off" /></label>
      <label className="search-field"><span>First day</span><input type="date" value={startDate} onChange={(event) => setStartDate(event.target.value)} /></label>
      <label className="search-field search-days"><span>How long?</span><select value={days} onChange={(event) => setDays(event.target.value)}>{Array.from({ length: 14 }, (_, index) => <option key={index + 1} value={index + 1}>{index + 1} {index === 0 ? "day" : "days"}</option>)}</select></label>
      <button className="button search-submit" type="submit">Find places <span aria-hidden="true">↗</span></button>
      {message ? <span className="form-hint" role="alert">{message}</span> : null}
    </form>
  );
}
