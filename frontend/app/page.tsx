"use client"
import { Box, Center, Container, Title } from "@mantine/core";
import useAuthStore from "./store/useAuthStore";
import { LandingHero } from "./components/landing/Hero";
import ContinueWatching from "./components/landing/ContinueWatching";
import RecentlyArchived from "./components/landing/RecentlyArchived";
import { useEffect } from "react";
import { useTranslations } from "next-intl";

export default function Home() {
  const { isLoggedIn } = useAuthStore();

  useEffect(() => {
    document.title = "DuckVOD";
  }, []);

  const t = useTranslations("HomePage");

  return (
    <div>
      {!isLoggedIn && (
        <Box mb={5}>
          <LandingHero />
        </Box>
      )}

      {isLoggedIn && (
        <Box>
          <Center>
            <Title>{t('continueWatching')}</Title>
          </Center>
          <Container mt={10} size={"7xl"}>
            <ContinueWatching count={4} />
          </Container>
        </Box>
      )}

      <Box>
        <Center>
          <Title>{t('recentlyArchived')}</Title>
        </Center>
        <Container mt={10} size={"7xl"}>
          <RecentlyArchived count={8} />
        </Container>
      </Box>



    </div>
  );
}
