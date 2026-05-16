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

type PrettyOpts = {
  base?: 1000 | 1024; // default 1000
  maxUnit?: "K" | "M" | "B" | "T"; // default "T"
};

// Pretty-print large numbers: 1500 -> "1.5K", 2000000 -> "2M", etc.
export function prettyNumber(n: number, opts: PrettyOpts = {}): string {
  const base = opts.base ?? 1000;
  const units = ["", "K", "M", "B", "T"];
  const maxIdx = Math.min(units.indexOf(opts.maxUnit ?? "T"), units.length - 1);
  if (!Number.isFinite(n)) return String(n);

  const sign = n < 0 ? "-" : "";
  let v = Math.abs(n);
  let u = 0;

  while (v >= base && u < maxIdx) {
    v /= base;
    u++;
  }

  // Truncate (not round): keep 1 decimal for <100 of a unit, else no decimals
  const factor = v < 100 ? 10 : 1;
  const truncated = Math.trunc(v * factor) / factor;

  // Strip trailing ".0"
  const str = (
    factor === 10 ? truncated.toFixed(1) : truncated.toFixed(0)
  ).replace(/\.0$/, "");

  return sign + str + units[u];
}

// durationToTime converts the provided video duration in seconds to 'HH:mm:ss'
// dayjs.duration doesn't work well with longer >=24 hour durations
export function durationToTime(seconds: number) {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;

  const formattedHours = String(hours).padStart(2, '0');
  const formattedMinutes = String(minutes).padStart(2, '0');
  const formattedSeconds = String(secs).padStart(2, '0');

  return `${formattedHours}:${formattedMinutes}:${formattedSeconds}`;
}