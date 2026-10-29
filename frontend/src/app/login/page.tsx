import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { AuthForm } from "@/components/auth-form";
import { SiteHeader } from "@/components/site-header";
import Paper from "@mui/material/Paper";

export const metadata: Metadata = { title: "Sign in" };
export default function LoginPage() {
  return <main className="auth-shell"><SiteHeader /><section className="auth-layout"><Paper className="auth-panel" variant="outlined" elevation={0}><Link href="/" className="back-link">← Home</Link><span className="eyebrow">Welcome back</span><h1>Good to see you.</h1><p className="auth-lede">Sign in to pick up where you left off.</p><Suspense fallback={<div className="loading-line">Loading sign-in…</div>}><AuthForm mode="login" /></Suspense></Paper></section></main>;
}
