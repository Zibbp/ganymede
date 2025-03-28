import '@mantine/core/styles.layer.css';
import '@mantine/notifications/styles.layer.css';
import '@mantine/carousel/styles.layer.css';
import '@mantine/charts/styles.layer.css';
import 'mantine-datatable/styles.layer.css';
import '@/app/global.css'

import { ColorSchemeScript } from '@mantine/core';
import type { Metadata } from "next";
import Providers from './providers';
import { EnvScript, PublicEnvScript } from 'next-runtime-env';
import { getLocale } from 'next-intl/server';
import { NextIntlClientProvider } from 'next-intl';

export const metadata: Metadata = {
  title: "Ganymede",
  description: "A platform to archive live streams and videos.",
};

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {

  const locale = await getLocale()

  return (
    <html lang={locale} suppressHydrationWarning>
      <head>
        <PublicEnvScript />
        <EnvScript
          env={{
            NEXT_PUBLIC_SHOW_SSO_LOGIN_BUTTON: process.env.SHOW_SSO_LOGIN_BUTTON,
            NEXT_PUBLIC_FORCE_SSO_AUTH: process.env.FORCE_SSO_AUTH,
            NEXT_PUBLIC_REQUIRE_LOGIN: process.env.REQUIRE_LOGIN,
            NEXT_PUBLIC_API_URL: process.env.API_URL,
            NEXT_PUBLIC_CDN_URL: process.env.CDN_URL,
          }}
        />
        <ColorSchemeScript defaultColorScheme='dark' />
      </head>
      <body>

        <Providers>
          <NextIntlClientProvider>
            {children}
          </NextIntlClientProvider>
        </Providers>

      </body>
    </html>
  );
}
