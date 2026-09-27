export type VideoNoteTimestamp = {
  id: string;
  label: string;
  time: number;
};

export function parseTimestampHref(href?: string): number | null {
  if (!href) return null;

  const hashMatch = href.match(/#t=(\d+)/);
  if (hashMatch) return parseInt(hashMatch[1], 10);

  // Also handle pasted share links (/videos/<id>?t=123 or full URLs).
  const queryMatch = href.match(/[?&]t=(\d+)/);
  if (queryMatch) return parseInt(queryMatch[1], 10);

  return null;
}

function cleanMarkdownLabel(label: string) {
  return label
    .replace(/\\([\\`*_[\]{}()#+.!~-])/g, '$1')
    .replace(/[*_~`]/g, '')
    .trim();
}

export function extractVideoNoteTimestamps(notes?: string): VideoNoteTimestamp[] {
  if (!notes) return [];

  const timestamps: VideoNoteTimestamp[] = [];
  const seenTimes = new Set<number>();
  const markdownLinkPattern = /\[([^\]]*)\]\(([^)\s]+)(?:\s+["'][^"']*["'])?\)/g;

  for (const match of notes.matchAll(markdownLinkPattern)) {
    const time = parseTimestampHref(match[2]);
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
