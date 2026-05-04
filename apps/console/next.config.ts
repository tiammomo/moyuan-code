import type { NextConfig } from "next";

const backendOrigin = process.env.MOYUAN_API_ORIGIN ?? "http://127.0.0.1:8080";

const nextConfig: NextConfig = {
  cacheComponents: true,
  devIndicators: false,
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${backendOrigin}/v1/:path*`,
      },
      {
        source: "/healthz",
        destination: `${backendOrigin}/healthz`,
      },
    ];
  },
};

export default nextConfig;
