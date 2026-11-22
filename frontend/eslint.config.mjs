import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

export default defineConfig([
  ...nextVitals,
  ...nextTs,
  globalIgnores([".next/**", "out/**", "build/**", "next-env.d.ts"]),
  {
    // Plain CommonJS Node utility (run via `node frontend/translation-coverage.js`),
    // not part of the bundled TypeScript app.
    files: ["translation-coverage.js"],
    rules: {
      "@typescript-eslint/no-require-imports": "off",
    },
  },
]);
