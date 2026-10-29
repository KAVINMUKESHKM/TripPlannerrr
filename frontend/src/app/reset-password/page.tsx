import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { ResetPasswordForm } from "@/components/password-reset-form";
import { SiteHeader } from "@/components/site-header";
import Paper from "@mui/material/Paper";

export const metadata: Metadata = { title: "Choose a new password" };

export default function ResetPasswordPage() {
  return <main className="auth-shell"><SiteHeader /><section className="auth-layout"><Paper className="auth-panel" variant="outlined" elevation={0}><Link href="/login" className="back-link">← Sign in</Link><span className="eyebrow">Account recovery</span><h1>Choose a new password.</h1><p className="auth-lede">Reset links expire after 30 minutes and can only be used once.</p><Suspense fallback={<div className="loading-line">Loading reset form…</div>}><ResetPasswordForm /></Suspense></Paper></section></main>;
}
