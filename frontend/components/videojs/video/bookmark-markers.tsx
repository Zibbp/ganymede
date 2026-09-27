'use client';

import { usePlayer } from '@videojs/react/video';
import { useTranslations } from 'next-intl';

export type TimelineBookmark = {
  id: string;
  label: string;
  time: number;
};

function formatTime(seconds: number) {
  const roundedSeconds = Math.round(seconds);
  const hours = Math.floor(roundedSeconds / 3600);
  const minutes = Math.floor((roundedSeconds % 3600) / 60);
  const remainingSeconds = roundedSeconds % 60;

  return hours > 0
    ? `${hours}:${minutes.toString().padStart(2, '0')}:${remainingSeconds.toString().padStart(2, '0')}`
    : `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
}

function parseTimeLabel(label: string) {
  if (!/^\d{1,}:\d{2}:\d{2}$/.test(label)) return null;

  const [hours, minutes, seconds] = label.split(':').map(Number);
  return hours * 3600 + minutes * 60 + seconds;
}

function formatBookmarkLabel(label: string, time: number) {
  const formattedTime = formatTime(time);
  const describedTimestamp = label.match(/^(.*?)\s+[—–-]\s+(\d{1,}:\d{2}:\d{2})$/);

  if (describedTimestamp && parseTimeLabel(describedTimestamp[2]) === time) {
    const description = describedTimestamp[1].trim();
    return description ? `${description} (${formattedTime})` : formattedTime;
  }

  return label && parseTimeLabel(label) !== time
    ? `${label} (${formattedTime})`
    : formattedTime;
}

export interface BookmarkMarkersProps {
  bookmarks: readonly TimelineBookmark[];
}

export function BookmarkMarkers({ bookmarks }: BookmarkMarkersProps) {
  const t = useTranslations('VideoComponents');
  const player = usePlayer();
  const duration = usePlayer((state) => (state as unknown as { duration: number }).duration);

  if (!Number.isFinite(duration) || duration <= 0 || bookmarks.length === 0) {
    return null;
  }

  return (
    <div className="video-bookmark-markers">
      {bookmarks.map((bookmark) => {
        if (bookmark.time < 0 || bookmark.time > duration) return null;

        const label = formatBookmarkLabel(bookmark.label, bookmark.time);

        return (
          <button
            key={bookmark.id}
            type="button"
            className="video-bookmark-marker"
            style={{ left: `${(bookmark.time / duration) * 100}%` }}
            data-label={label}
            aria-label={`${t('notesJumpToTimestamp')}: ${label}`}
            onClick={() => player.seek(bookmark.time)}
          />
        );
      })}
    </div>
  );
}
