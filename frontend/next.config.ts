import type { NextConfig } from "next";
const nextConfig: NextConfig = {
  reactStrictMode: true,
  turbopack: { root: process.cwd() },
  experimental: { useTypeScriptCli: false },
  async rewrites() {
    const apiOrigin = (process.env.API_INTERNAL_URL ?? "http://localhost:8080").replace(/\/+$/, "");
    return [{ source: "/api/:path*", destination: `${apiOrigin}/api/:path*` }];
  },
};
export default nextConfig;
