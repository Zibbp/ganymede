import { Hotkey } from '@videojs/react';

import { LivePlaybackHotkeys } from './live-playback-hotkeys';

export interface LiveVideoHotkeysProps {
  disabled?: boolean | undefined;
}

export function LiveVideoHotkeys({ disabled = false }: LiveVideoHotkeysProps = {}) {
  return (
    <>
      <LivePlaybackHotkeys disabled={disabled} />
      <Hotkey disabled={disabled} keys="f" action="toggleFullscreen" />
      <Hotkey disabled={disabled} keys="c" action="toggleSubtitles" />
      <Hotkey disabled={disabled} keys="i" action="togglePictureInPicture" />
    </>
  );
}
