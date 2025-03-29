"use server";

import { cookies } from "next/headers";

const COOKIE_NAME = "NEXT_LOCALE";

export async function getUserLocale() {
  // If the default locale is set in the environment, use it
  if (
    process.env.NEXT_PUBLIC_DEFAULT_LOCALE &&
    process.env.NEXT_PUBLIC_DEFAULT_LOCALE !== "undefined"
  ) {
    return process.env.NEXT_PUBLIC_DEFAULT_LOCALE;
  }
  // Otherwise, use the locale from the cookie
  return (await cookies()).get(COOKIE_NAME)?.value || "en";
}

export async function setUserLocale(locale: string) {
  return (await cookies()).set(COOKIE_NAME, locale);
}
