"use client";

import Link from "next/link";
import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { currentUser } from "@/lib/api";

function CallbackContent() {
  const router = useRouter();
  const params = useSearchParams();
  const [message, setMessage] = useState("Finishing your sign-in…");
  const cancelled = params.get("error") === "cancelled";
  useEffect(() => {
    if (cancelled) { setMessage("Google sign-in was cancelled. You can try again whenever you’re ready."); return; }
    let live = true;
    currentUser().then(() => { if (live) { router.replace("/trips"); router.refresh(); } }).catch(() => { if (live) setMessage("We couldn’t confirm your sign-in. Please try again."); });
    return () => { live = false; };
  }, [cancelled, router]);
  return <section className="callback-card"><span className="eyebrow">Trip Planner</span><h1>{cancelled ? "No rush." : "One moment."}</h1><p role="status">{message}</p>{cancelled || message.startsWith("We couldn’t") ? <Link className="button" href="/login">Back to sign in</Link> : <span className="loading-dot" aria-hidden="true" />}</section>;
}

export default function AuthCallbackPage() {
  return <main className="callback-shell"><Suspense fallback={<p>Finishing your sign-in…</p>}><CallbackContent /></Suspense></main>;
}
