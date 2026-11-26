import { Hotkey } from '@videojs/react';

export interface LivePlaybackHotkeysProps {
  disabled?: boolean | undefined;
}

export function LivePlaybackHotkeys({ disabled = false }: LivePlaybackHotkeysProps = {}) {
  return (
    <>
      <Hotkey disabled={disabled} keys="Space" action="togglePaused" />
      <Hotkey disabled={disabled} keys="k" action="togglePaused" />
      <Hotkey disabled={disabled} keys="m" action="toggleMuted" />
      {/* Seek keys match the VOD player. */}
      <Hotkey disabled={disabled} keys="ArrowRight" action="seekStep" />
      <Hotkey disabled={disabled} keys="ArrowLeft" action="seekStep" />
      <Hotkey disabled={disabled} keys="l" action="seekStep" />
      <Hotkey disabled={disabled} keys="j" action="seekStep" />
      <Hotkey disabled={disabled} keys="ArrowUp" action="volumeStep" />
      <Hotkey disabled={disabled} keys="ArrowDown" action="volumeStep" />
      <Hotkey disabled={disabled} keys="0-9" action="seekToPercent" />
      {/* No End key; LIVE returns to the live edge. */}
      <Hotkey disabled={disabled} keys="Home" action="seekToPercent" value={0} />
      {/* Playback speed. */}
      <Hotkey disabled={disabled} keys=">" action="speedUp" />
      <Hotkey disabled={disabled} keys="<" action="speedDown" />
    </>
  );
}
