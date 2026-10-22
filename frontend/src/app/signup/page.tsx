import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { AuthForm } from "@/components/auth-form";
import { SiteHeader } from "@/components/site-header";

export const metadata: Metadata = { title: "Create an account" };
export default function SignupPage() {
  return <main className="auth-shell"><SiteHeader /><section className="auth-layout"><div className="auth-photo signup-photo"><div className="auth-photo-note"><span>Take the long way</span><strong>once in a while.</strong></div></div><div className="auth-panel"><Link href="/" className="back-link">← Back to exploring</Link><span className="eyebrow">A place for your plans</span><h1>Keep the good<br />days close.</h1><p className="auth-lede">Create an account to save and shape your itineraries.</p><Suspense fallback={<div className="loading-line">Loading sign-up…</div>}><AuthForm mode="signup" /></Suspense></div></section></main>;
}
