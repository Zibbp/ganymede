import '@mantine/core/styles.layer.css';
import '@mantine/notifications/styles.layer.css';
import '@mantine/carousel/styles.layer.css';
import 'mantine-datatable/styles.layer.css';
import '@/app/global.css'

import { ColorSchemeScript } from '@mantine/core';
import type { Metadata } from "next";
import Providers from './providers';
import { PublicEnvScript } from 'next-runtime-env';

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
