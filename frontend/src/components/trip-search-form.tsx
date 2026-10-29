"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import Button from "@mui/material/Button";
import MenuItem from "@mui/material/MenuItem";
import TextField from "@mui/material/TextField";

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
      <div className="search-field"><label htmlFor="home-destination">Where to?</label><TextField id="home-destination" className="search-destination" value={destination} onChange={(event) => { setDestination(event.target.value); setMessage(""); }} placeholder="A city, coast, or mountain town" autoComplete="off" /></div>
      <div className="search-field"><label htmlFor="home-date">First day</label><TextField id="home-date" type="date" value={startDate} onChange={(event) => setStartDate(event.target.value)} /></div>
      <div className="search-field"><label htmlFor="home-days">How long?</label><TextField id="home-days" select value={days} onChange={(event) => setDays(event.target.value)}>{Array.from({ length: 14 }, (_, index) => <MenuItem key={index + 1} value={index + 1}>{index + 1} {index === 0 ? "day" : "days"}</MenuItem>)}</TextField></div>
      <Button className="search-submit" variant="contained" type="submit">Find places <span aria-hidden="true">↗</span></Button>
      {message ? <span className="form-hint" role="alert">{message}</span> : null}
    </form>
  );
}
