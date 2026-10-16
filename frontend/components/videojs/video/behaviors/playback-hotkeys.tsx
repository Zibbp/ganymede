import { Hotkey } from '@videojs/react';

export interface PlaybackHotkeysProps {
  disabled?: boolean | undefined;
}

export function PlaybackHotkeys({ disabled = false }: PlaybackHotkeysProps = {}) {
  return (
    <>
      <Hotkey disabled={disabled} keys="Space" action="togglePaused" />
      <Hotkey disabled={disabled} keys="k" action="togglePaused" />
      <Hotkey disabled={disabled} keys="m" action="toggleMuted" />
      <Hotkey disabled={disabled} keys="ArrowRight" action="seekStep" />
      <Hotkey disabled={disabled} keys="ArrowLeft" action="seekStep" />
      <Hotkey disabled={disabled} keys="l" action="seekStep" />
      <Hotkey disabled={disabled} keys="j" action="seekStep" />
      <Hotkey disabled={disabled} keys="ArrowUp" action="volumeStep" />
      <Hotkey disabled={disabled} keys="ArrowDown" action="volumeStep" />
      <Hotkey disabled={disabled} keys="0-9" action="seekToPercent" />
      <Hotkey disabled={disabled} keys="Home" action="seekToPercent" value={0} />
      <Hotkey disabled={disabled} keys="End" action="seekToPercent" value={100} />
      <Hotkey disabled={disabled} keys=">" action="speedUp" />
      <Hotkey disabled={disabled} keys="<" action="speedDown" />
    </>
  );
}
