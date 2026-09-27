export type VideoNoteTimestamp = {
  id: string;
  label: string;
  time: number;
};

export function parseTimestampHref(href: string | undefined, currentVideoId: string): number | null {
  if (!href) return null;

  const hashMatch = href.match(/^#t=(\d+)$/);
  if (hashMatch) return parseInt(hashMatch[1], 10);

  // Also handle pasted share links (/videos/<id>?t=123 or full URLs), but
  // only when they point to the video whose notes are being displayed.
  try {
    const url = new URL(href, "http://ganymede.local");
    const videoPathMatch = url.pathname.match(/\/videos\/([^/]+)\/?$/);
    if (!videoPathMatch || decodeURIComponent(videoPathMatch[1]) !== currentVideoId) return null;

    const time = url.searchParams.get("t");
    if (time && /^\d+$/.test(time)) return parseInt(time, 10);
  } catch {
    return null;
  }

  return null;
}

function cleanMarkdownLabel(label: string) {
  return label
    .replace(/\\([\\`*_[\]{}()#+.!~-])/g, '$1')
    .replace(/[*_~`]/g, '')
    .trim();
}

export function extractVideoNoteTimestamps(notes: string | undefined, currentVideoId: string): VideoNoteTimestamp[] {
  if (!notes) return [];

  const timestamps: VideoNoteTimestamp[] = [];
  const seenTimes = new Set<number>();
  const markdownLinkPattern = /\[([^\]]*)\]\(([^)\s]+)(?:\s+["'][^"']*["'])?\)/g;

  for (const match of notes.matchAll(markdownLinkPattern)) {
    const time = parseTimestampHref(match[2], currentVideoId);
    if (time === null || seenTimes.has(time)) continue;

    seenTimes.add(time);
    timestamps.push({
      id: `note-${time}`,
      label: cleanMarkdownLabel(match[1]),
      time,
    });
  }

  return timestamps.sort((a, b) => a.time - b.time);
}
