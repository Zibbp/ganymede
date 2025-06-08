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
import ForceLogin from './components/authentication/ForceLogin';

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
            NEXT_PUBLIC_SHOW_LOCALE_BUTTON: process.env.SHOW_LOCALE_BUTTON,
            NEXT_PUBLIC_DEFAULT_LOCALE: process.env.DEFAULT_LOCALE,
            NEXT_PUBLIC_FORCE_LOGIN: process.env.FORCE_LOGIN,
          }}
        />
        <ColorSchemeScript defaultColorScheme='dark' />
      </head>
      <body>

        <NextIntlClientProvider>
          <Providers>
            {/* ForceLogin prevents rendering the rest of the page if login is required */}
            <ForceLogin>{children}</ForceLogin>
          </Providers>
        </NextIntlClientProvider>

      </body>
    </html>
  );
}
