"use client";

import Link from "next/link";
import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { ApiError, googleSignInURL, login, register } from "@/lib/api";
import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import Divider from "@mui/material/Divider";
import TextField from "@mui/material/TextField";

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
      {isSignup ? <TextField required fullWidth label="Your name" autoComplete="name" value={name} onChange={(event) => setName(event.target.value)} /> : null}
      <TextField required fullWidth label="Email address" type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} />
      <TextField required fullWidth label="Password" type="password" helperText="Use at least 8 characters." slotProps={{ htmlInput: { minLength: 8 } }} autoComplete={isSignup ? "new-password" : "current-password"} value={password} onChange={(event) => setPassword(event.target.value)} />
      {error ? <Alert severity="error" role="alert">{error}</Alert> : null}
      <Button className="auth-submit" type="submit" variant="contained" size="large" disabled={busy}>{busy ? "Please wait…" : isSignup ? "Create your account" : "Sign in"}</Button>
      {!isSignup ? <p className="auth-switch"><Link href="/forgot-password">Forgot your password?</Link></p> : null}
      <Divider>or continue with</Divider>
      <Button className="auth-submit" component="a" href={googleSignInURL} variant="outlined" size="large">Continue with Google</Button>
      <p className="auth-switch">{isSignup ? "Already have an account?" : "New to Trip Planner?"} <Link href={isSignup ? "/login" : "/signup"}>{isSignup ? "Sign in" : "Create an account"}</Link></p>
    </form>
  );
}
