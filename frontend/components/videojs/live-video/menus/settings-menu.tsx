import type { ComponentProps } from 'react';

import { AudioTrackMenu } from '@/components/videojs/ui/audio-track-menu';
import { PlaybackRateSubmenu } from '@/components/videojs/ui/playback-rate-submenu';
import { QualityMenu } from '@/components/videojs/ui/quality-menu';
import { SettingsMenu } from '@/components/videojs/ui/settings-menu';

export type LiveVideoSettingsMenuProps = Omit<NonNullable<ComponentProps<typeof SettingsMenu>>, 'children'>;

// Live settings; speed needs the player's playbackRate feature.
export function LiveVideoSettingsMenu(props: LiveVideoSettingsMenuProps = {}) {
  return (
    <SettingsMenu {...props}>
      <QualityMenu />
      <AudioTrackMenu />
      <PlaybackRateSubmenu />
    </SettingsMenu>
  );
}
