import { useCallback, useEffect, useRef, useState } from "react";
import { env } from "next-runtime-env";
import { escapeURL, playlistPathForVideo } from "@/app/util/util";

const PLAYLIST_HEADER = "#EXTM3U";
// A server that accepts the connection and then never answers must not leave
// the player without a source at all.
const PROBE_TIMEOUT_MS = 5000;

/**
 * usePlaylistSource reports the playlist that belongs to a video, or null when
 * it has none.
 *
 * A playlist sits next to the video under the same name and addresses it by
 * byte ranges. Reading the layout from there is what lets long recordings start
 * on Apple devices, where the sample table of a multi-hour MP4 is large enough
 * that playback never begins. Videos archived before playlists existed, and
 * installations that turned them off, have none, so the video file itself stays
 * the fallback.
 *
 * The result is undefined until the check has run, so a caller can wait instead
 * of loading the video file and switching away from it a moment later.
 */
export function usePlaylistSource(
  videoPath: string,
  processing: boolean,
): { playlistUrl: string | null | undefined; dismissPlaylist: () => void } {
  const [playlistUrl, setPlaylistUrl] = useState<string | null | undefined>(undefined);
  const dismissed = useRef(false);

  useEffect(() => {
    dismissed.current = false;
    setPlaylistUrl(undefined);

    const playlistPath = processing ? null : playlistPathForVideo(videoPath);
    if (playlistPath == null) {
      setPlaylistUrl(null);
      return;
    }

    const candidate = `${env("NEXT_PUBLIC_CDN_URL") ?? ""}${escapeURL(playlistPath)}`;
    const abort = new AbortController();
    let superseded = false;
    const timeout = setTimeout(() => abort.abort(), PROBE_TIMEOUT_MS);

    // A proxy that answers a missing file with its own page would pass a plain
    // status check, so the response has to look like a playlist. Only the first
    // chunk is read and the rest is cancelled: a playlist of a long recording
    // runs to a hundred kilobytes, and the player fetches it again in full a
    // moment later anyway.
    fetch(candidate, { signal: abort.signal })
      .then(async (response) => {
        if (!response.ok || !response.body) return "";

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let head = "";
        try {
          // Chunks can be smaller than the line being looked for, so read on
          // until there is enough to judge.
          while (head.length < PLAYLIST_HEADER.length) {
            const { value, done } = await reader.read();
            if (done) break;
            head += decoder.decode(value, { stream: true });
          }
          return head;
        } finally {
          await reader.cancel().catch(() => undefined);
        }
      })
      .then((head) => {
        if (!superseded && !dismissed.current) setPlaylistUrl(head.startsWith(PLAYLIST_HEADER) ? candidate : null);
      })
      .catch(() => {
        // Keep playing the video file itself, unless this has already been
        // replaced by a check for another video.
        if (!superseded) setPlaylistUrl(null);
      })
      .finally(() => clearTimeout(timeout));

    return () => {
      superseded = true;
      clearTimeout(timeout);
      abort.abort();
    };
  }, [videoPath, processing]);

  // A playlist addresses the video by byte offset, so a video that was
  // modified after the playlist was written leaves the offsets pointing at
  // something else. That cannot be told apart from a correct playlist by
  // reading it, only by playback failing, so a failure gives up on the playlist
  // and plays the video file instead.
  const dismissPlaylist = useCallback(() => {
    dismissed.current = true;
    setPlaylistUrl(null);
  }, []);

  return { playlistUrl, dismissPlaylist };
}
