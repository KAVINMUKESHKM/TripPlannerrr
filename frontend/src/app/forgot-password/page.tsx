import type { Metadata } from "next";
import Link from "next/link";
import { ForgotPasswordForm } from "@/components/password-reset-form";
import { SiteHeader } from "@/components/site-header";
import Paper from "@mui/material/Paper";

export const metadata: Metadata = { title: "Reset your password" };

export default function ForgotPasswordPage() {
  return <main className="auth-shell"><SiteHeader /><section className="auth-layout"><Paper className="auth-panel" variant="outlined" elevation={0}><Link href="/login" className="back-link">← Sign in</Link><span className="eyebrow">Account recovery</span><h1>Forgot your password?</h1><p className="auth-lede">Enter your email and we’ll send a single-use reset link.</p><ForgotPasswordForm /></Paper></section></main>;
}
