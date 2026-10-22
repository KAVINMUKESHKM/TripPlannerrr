import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { AuthForm } from "@/components/auth-form";
import { SiteHeader } from "@/components/site-header";

export const metadata: Metadata = { title: "Sign in" };
export default function LoginPage() {
  return <main className="auth-shell"><SiteHeader /><section className="auth-layout"><div className="auth-photo"><div className="auth-photo-note"><span>Good plans leave room</span><strong>for the detour.</strong></div></div><div className="auth-panel"><Link href="/" className="back-link">← Back to exploring</Link><span className="eyebrow">Welcome back</span><h1>Your next trip<br />is waiting.</h1><p className="auth-lede">Sign in to pick up where you left off.</p><Suspense fallback={<div className="loading-line">Loading sign-in…</div>}><AuthForm mode="login" /></Suspense></div></section></main>;
}
