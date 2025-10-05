import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "DuckVOD",
    short_name: "DuckVOD",
    description: "Eine Plattform zum Archivieren von Live-Streams und Videos",
    start_url: "/",
    display: "standalone",
    background_color: "#141417",
    theme_color: "#000000",
    icons: [
      {
        src: "/android-chrome-192x192.png",
        sizes: "192x192",
        type: "image/png",
      },
      {
        src: "/android-chrome-512x512.png",
        sizes: "512x512",
        type: "image/png",
      },
    ],
  };
}
