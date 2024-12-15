"use client"
import { Box, Center, Container, Title } from "@mantine/core";
import useAuthStore from "./store/useAuthStore";
import { LandingHero } from "./components/landing/Hero";
import ContinueWatching from "./components/landing/ContinueWatching";
import RecentlyArchived from "./components/landing/RecentlyArchived";

export default function Home() {
  const { isLoggedIn } = useAuthStore();

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
            <Title>Continue Watching</Title>
          </Center>
          <Container mt={10} size={"7xl"}>
            <ContinueWatching count={4} />
          </Container>
        </Box>
      )}

      <Box>
        <Center>
          <Title>Recently Archived</Title>
        </Center>
        <Container mt={10} size={"7xl"}>
          <RecentlyArchived count={8} />
        </Container>
      </Box>



    </div>
  );
}
