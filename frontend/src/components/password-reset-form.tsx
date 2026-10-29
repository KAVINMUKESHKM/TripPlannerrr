"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import { ApiError, requestPasswordReset, resetPassword } from "@/lib/api";

export function ForgotPasswordForm() {
  const [email, setEmail] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError(""); setMessage("");
    try { setMessage((await requestPasswordReset(email)).message); }
    catch (cause) { setError(cause instanceof ApiError ? cause.message : "The service could not be reached."); }
    finally { setBusy(false); }
  }

  return <form className="auth-form" onSubmit={submit}>
    <TextField required fullWidth type="email" label="Email address" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} />
    {error ? <Alert severity="error" role="alert">{error}</Alert> : null}
    {message ? <Alert severity="success" role="status">{message}</Alert> : null}
    <Button className="auth-submit" type="submit" variant="contained" size="large" disabled={busy}>{busy ? "Sending…" : "Send reset link"}</Button>
    <p className="auth-switch"><Link href="/login">Back to sign in</Link></p>
  </form>;
}

export function ResetPasswordForm() {
  const params = useSearchParams();
  const [token] = useState(() => params.get("token") ?? "");
  const [password, setPassword] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (token) window.history.replaceState(null, "", "/reset-password");
  }, [token]);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError(""); setMessage("");
    try { await resetPassword(token, password); setMessage("Your password has been changed. Sign in again on your devices."); }
    catch (cause) { setError(cause instanceof ApiError ? cause.message : "The service could not be reached."); }
    finally { setBusy(false); }
  }

  return <form className="auth-form" onSubmit={submit}>
    {!token ? <Alert severity="error" role="alert">This reset link is missing or invalid.</Alert> : null}
    <TextField required fullWidth label="New password" type="password" autoComplete="new-password" helperText="Use 8 to 72 characters." slotProps={{ htmlInput: { minLength: 8, maxLength: 72 } }} value={password} onChange={(event) => setPassword(event.target.value)} />
    {error ? <Alert severity="error" role="alert">{error}</Alert> : null}
    {message ? <Alert severity="success" role="status">{message}</Alert> : null}
    <Button className="auth-submit" type="submit" variant="contained" size="large" disabled={busy || !token || !!message}>{busy ? "Saving…" : "Set new password"}</Button>
    <p className="auth-switch"><Link href="/login">Back to sign in</Link></p>
  </form>;
}
