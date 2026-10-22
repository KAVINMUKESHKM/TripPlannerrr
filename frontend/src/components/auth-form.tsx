"use client";

import Link from "next/link";
import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { ApiError, googleSignInURL, login, register } from "@/lib/api";

export function AuthForm({ mode }: { mode: "login" | "signup" }) {
  const router = useRouter();
  const params = useSearchParams();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const isSignup = mode === "signup";

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setError(""); setBusy(true);
    try {
      if (isSignup) await register({ email, password, full_name: name.trim() });
      else await login({ email, password });
      const returnTo = params.get("next");
      const safeReturnTo = returnTo?.startsWith("/") && !returnTo.startsWith("//") && !returnTo.includes("\\") ? returnTo : "/trips";
      router.replace(safeReturnTo);
      router.refresh();
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : "The service could not be reached. Check that the API is running.");
    } finally { setBusy(false); }
  }

  return (
    <form className="auth-form" onSubmit={submit}>
      {isSignup ? <label className="form-label">Your name<input required autoComplete="name" value={name} onChange={(event) => setName(event.target.value)} /></label> : null}
      <label className="form-label">Email address<input required type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} /></label>
      <label className="form-label">Password<input required type="password" minLength={8} autoComplete={isSignup ? "new-password" : "current-password"} value={password} onChange={(event) => setPassword(event.target.value)} /><span className="field-note">Use at least 8 characters.</span></label>
      {error ? <p className="form-error" role="alert">{error}</p> : null}
      <button className="button auth-submit" disabled={busy}>{busy ? "Please wait…" : isSignup ? "Create your account" : "Sign in"}</button>
      <div className="auth-divider"><span>or continue with</span></div>
      <a className="button button-outline auth-submit" href={googleSignInURL}>Continue with Google</a>
      <p className="auth-switch">{isSignup ? "Already have an account?" : "New to Trip Planner?"} <Link href={isSignup ? "/login" : "/signup"}>{isSignup ? "Sign in" : "Create an account"}</Link></p>
    </form>
  );
}
