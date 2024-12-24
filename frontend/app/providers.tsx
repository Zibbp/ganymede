'use client'
import { QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { getQueryClient } from '@/app/get-query-client'
import type * as React from 'react'
import { Container, createTheme, MantineProvider, rem } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { Navbar } from './layout/Navbar';
import useSettingsStore from './store/useSettingsStore';
import { useEffect } from 'react';
import useAuthStore from './store/useAuthStore'
import localFont from 'next/font/local'
const localInterFont = localFont({
  src: "./Inter-Variable.ttf"
})

const CONTAINER_SIZES: Record<string, string> = {
  xxs: rem(300),
  xs: rem(400),
  sm: rem(500),
  md: rem(600),
  lg: rem(700),
  xl: rem(800),
  xxl: rem(900),
  "3xl": rem(1000),
  "4xl": rem(1100),
  "5xl": rem(1200),
  "6xl": rem(1300),
  "7xl": rem(1400),
};

const theme = createTheme({
  fontFamily: localInterFont.style.fontFamily,
  breakpoints: {
    xs: "30em",
    sm: "48em",
    md: "64em",
    lg: "74em",
    xl: "90em",
    xxl: "100em",
    "3xl": "116em",
    "4xl": "130em",
    "5xl": "146em",
    "6xl": "160em"
  },
  components: {
    Container: Container.extend({
      vars: (_, { size, fluid }) => ({
        root: {
          '--container-size': fluid
            ? '100%'
            : size !== undefined && size in CONTAINER_SIZES
              ? CONTAINER_SIZES[size]
              : rem(size),
        },
      }),
    }),
  },
  colors: {
    dark: [
      '#C1C2C5',
      '#A6A7AB',
      '#909296',
      '#5c5f66',
      '#373A40',
      '#2C2E33',
      '#18181C',
      '#141417',
      '#141517',
      '#101113',
    ],
  },
});

export default function Providers({ children }: { children: React.ReactNode }) {
  const { fetchUser } = useAuthStore()
  const queryClient = getQueryClient()

  const videoTheaterMode = useSettingsStore((state) => state.videoTheaterMode);

  // Attempt to authenticate user on initial page load
  useEffect(() => {
    fetchUser()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <MantineProvider defaultColorScheme="dark" theme={theme}>
      <Notifications />
      <QueryClientProvider client={queryClient}>
        {!videoTheaterMode && <Navbar />}
        {children}
        <ReactQueryDevtools />
      </QueryClientProvider>
    </MantineProvider>
  )
}
