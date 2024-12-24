import type { NextConfig } from "next";
import { makeEnvPublic } from "next-runtime-env";

makeEnvPublic([
  "API_URL",
  "CDN_URL",
  "SHOW_SSO_LOGIN_BUTTON",
  "FORCE_SSO_AUTH",
  "REQUIRE_LOGIN",
]);

const nextConfig: NextConfig = {
  output: "standalone",
};

export default nextConfig;
