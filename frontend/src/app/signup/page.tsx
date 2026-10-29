import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { AuthForm } from "@/components/auth-form";
import { SiteHeader } from "@/components/site-header";
import Paper from "@mui/material/Paper";

export const metadata: Metadata = { title: "Create an account" };
export default function SignupPage() {
  return <main className="auth-shell"><SiteHeader /><section className="auth-layout"><Paper className="auth-panel" variant="outlined" elevation={0}><Link href="/" className="back-link">← Home</Link><span className="eyebrow">Create your account</span><h1>Keep your trips close.</h1><p className="auth-lede">Save itineraries and pick them up whenever you’re ready.</p><Suspense fallback={<div className="loading-line">Loading sign-up…</div>}><AuthForm mode="signup" /></Suspense></Paper></section></main>;
}
