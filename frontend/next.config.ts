import type { NextConfig } from "next";
import { makeEnvPublic } from "next-runtime-env";
import createNextIntlPlugin from "next-intl/plugin";

makeEnvPublic([
  "API_URL",
  "CDN_URL",
  "SHOW_SSO_LOGIN_BUTTON",
  "FORCE_SSO_AUTH",
  "REQUIRE_LOGIN",
  "SHOW_LOCALE_BUTTON",
  "DEFAULT_LOCALE",
  "FORCE_LOGIN",
]);

const nextConfig: NextConfig = {
  output: "standalone",
};

const withNextIntl = createNextIntlPlugin();
export default withNextIntl(nextConfig);
