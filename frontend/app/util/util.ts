import { useEffect } from "react";

export function escapeURL(str: string): string {
  return str.replace(/#/g, "%23");
}

// saves an npm depedency - can't use the crypto API as that is not available over HTTP
// https://stackoverflow.com/questions/105034/how-do-i-create-a-guid-uuid
export function uuidv4() {
  return "10000000-1000-4000-8000-100000000000".replace(/[018]/g, (c) =>
    (
      +c ^
      (crypto.getRandomValues(new Uint8Array(1))[0] & (15 >> (+c / 4)))
    ).toString(16)
  );
}

export function usePageTitle(title: string) {
  useEffect(() => {
    document.title = title;
  }, [title]);
}

// https://stackoverflow.com/a/18650828
export function formatBytes(bytes: number, decimals = 2) {
  if (bytes <= 0) return "0 Bytes";

  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ["Bytes", "KiB", "MiB", "GiB", "TiB", "PiB"];

  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`;
}
