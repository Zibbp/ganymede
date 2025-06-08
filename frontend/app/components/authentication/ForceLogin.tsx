"use client";

import { useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";
import useAuthStore from "@/app/store/useAuthStore";

const requireLoginFull = process.env.NEXT_PUBLIC_FORCE_LOGIN === "true";

// ForceLogin component checks if the user is logged in and redirects to the login page if not
export default function ForceLogin({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const { isLoggedIn, isLoading } = useAuthStore();

  const isAuthPage = pathname === "/login" || pathname === "/register";

  useEffect(() => {
    if (requireLoginFull && !isLoading && !isLoggedIn && !isAuthPage) {
      router.replace("/login");
    }
  }, [isLoggedIn, isLoading, isAuthPage, router]);

  if (requireLoginFull && !isAuthPage) {
    if (isLoading) return null;
    if (!isLoggedIn) return null;
  }

  return <>{children}</>;
}
