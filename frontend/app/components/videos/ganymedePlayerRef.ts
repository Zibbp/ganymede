import type { RefObject } from "react";

// Platform-neutral player handle for the Video.js 10 migration.
// Vidstack's `MediaPlayerInstance` is gone; media components forward refs
// to the underlying `<video>` element, so consumers share an
// `HTMLVideoElement` ref plus these small helpers.
type GanymedePlayerRef = RefObject<HTMLVideoElement | null>;

const getPlayerTime = (videoEl: HTMLVideoElement | null): number => {
  if (!videoEl) return 0;
  const time = videoEl.currentTime;
  return Number.isFinite(time) ? time : 0;
};

const setPlayerTime = (
  videoEl: HTMLVideoElement | null,
  seconds: number,
): void => {
  if (!videoEl) return;
  if (!Number.isFinite(seconds)) return;
  videoEl.currentTime = Math.max(0, seconds);
};

const isPlayerPaused = (videoEl: HTMLVideoElement | null): boolean => {
  if (!videoEl) return true;
  return videoEl.paused;
};

const playPlayer = (videoEl: HTMLVideoElement | null): Promise<void> => {
  if (!videoEl) return Promise.resolve();
  try {
    const result = videoEl.play();
    if (result instanceof Promise) return result.catch(() => undefined);
  } catch {
    // play() can throw when the element is not ready; callers sync state via events.
  }
  return Promise.resolve();
};

const pausePlayer = (videoEl: HTMLVideoElement | null): void => {
  if (!videoEl) return;
  try {
    videoEl.pause();
  } catch {
    // Ignore pause errors during teardown.
  }
};

export type { GanymedePlayerRef };
export { getPlayerTime, setPlayerTime, isPlayerPaused, playPlayer, pausePlayer };
