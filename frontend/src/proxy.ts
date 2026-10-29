import { randomUUID } from "node:crypto";
import { NextResponse, type NextRequest } from "next/server";

export function proxy(request: NextRequest) {
  const nonce = Buffer.from(randomUUID()).toString("base64");
  const isDev = process.env.NODE_ENV === "development";
  let customStyleOrigin = "";
  try {
    const styleURL = process.env.NEXT_PUBLIC_OPENFREEMAP_STYLE_URL;
    if (styleURL) {
      const parsedStyleURL = new URL(styleURL);
      if (parsedStyleURL.protocol === "https:" || (isDev && parsedStyleURL.hostname === "localhost")) {
        customStyleOrigin = parsedStyleURL.origin;
      }
    }
  } catch {
    customStyleOrigin = "";
  }
  const mapSources = ["https://tiles.openfreemap.org", "https://fonts.openmaptiles.org", customStyleOrigin].filter(Boolean);
  const connectSources = ["'self'", ...mapSources]
    .filter(Boolean)
    .join(" ");
  const imageSources = ["'self'", "data:", "blob:", ...mapSources].join(" ");
  const fontSources = ["'self'", "data:", "https://fonts.openmaptiles.org", customStyleOrigin].filter(Boolean).join(" ");
  const policy = [
    "default-src 'self'",
    `script-src 'self' 'nonce-${nonce}' 'strict-dynamic'${isDev ? " 'unsafe-eval'" : ""}`,
    `style-src 'self' 'nonce-${nonce}' https://fonts.openmaptiles.org`,
    "style-src-attr 'unsafe-inline'",
    `img-src ${imageSources}`,
    `font-src ${fontSources}`,
    `connect-src ${connectSources}`,
    "worker-src 'self' blob:",
    "object-src 'none'",
    "base-uri 'self'",
    "frame-ancestors 'none'",
    "form-action 'self'",
  ].join("; ");

  const requestHeaders = new Headers(request.headers);
  requestHeaders.set("x-nonce", nonce);
  requestHeaders.set("Content-Security-Policy", policy);
  const response = NextResponse.next({ request: { headers: requestHeaders } });
  response.headers.set("Content-Security-Policy", policy);
  if (request.nextUrl.pathname === "/reset-password" || request.nextUrl.pathname === "/forgot-password") {
    response.headers.set("Cache-Control", "no-store");
  }
  return response;
}

export const config = {
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico).*)"],
};
