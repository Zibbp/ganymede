"use client"
import { Button } from "@mantine/core";
import useAuthStore from "./store/useAuthStore";
import { LandingHero } from "./components/landing/Hero";

export default function Home() {
  const { user, isLoggedIn } = useAuthStore();

  return (
    <div>
      <LandingHero />

      home page
      <Button variant="filled">Button</Button>
      {isLoggedIn && (
        user?.username
      )}
    </div>
  );
}
