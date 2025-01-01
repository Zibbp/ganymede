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

export const metadata: Metadata = {
  title: "Ganymede",
  description: "A platform to archive live streams and videos.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {

  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <PublicEnvScript />
        <EnvScript
          env={{
            NEXT_PUBLIC_SHOW_SSO_LOGIN_BUTTON: process.env.SHOW_SSO_LOGIN_BUTTON
          }}
        />
        <ColorSchemeScript defaultColorScheme='dark' />
      </head>
      <body>

        <Providers>
          {children}
        </Providers>

      </body>
    </html>
  );
}
