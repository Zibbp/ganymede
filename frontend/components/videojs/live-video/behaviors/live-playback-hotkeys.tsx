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
      <Hotkey disabled={disabled} keys="ArrowUp" action="volumeStep" />
      <Hotkey disabled={disabled} keys="ArrowDown" action="volumeStep" />
    </>
  );
}
