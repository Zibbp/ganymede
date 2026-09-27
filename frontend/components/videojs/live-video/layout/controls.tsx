import { Controls, Tooltip } from '@videojs/react';
import type { ReactNode } from 'react';

import { AirPlayButton } from '@/components/videojs/ui/airplay-button';
import { ButtonTooltip } from '@/components/videojs/ui/button-tooltip';
import { CaptionsMenu } from '@/components/videojs/ui/captions-menu';
import { CastButton } from '@/components/videojs/ui/cast-button';
import { FullscreenButton } from '@/components/videojs/ui/fullscreen-button';
import { LiveButton } from '@/components/videojs/ui/live-button';
import { PiPButton } from '@/components/videojs/ui/pip-button';
import { PlayButton } from '@/components/videojs/ui/play-button';
import { VolumePopover } from '@/components/videojs/ui/volume-popover';
import { cn } from '@/lib/utils';

export interface DefaultLiveVideoControlsProps {
  customControls?: ReactNode;
}

export function DefaultLiveVideoControls({ customControls }: DefaultLiveVideoControlsProps = {}) {
  return (
    <Controls.Root>
      <Controls.Backdrop className={'video-controls-backdrop'} />
      <Controls.Content className={cn('video-controls', 'video-controls-content', 'video-controls-spaced')}>
        <Tooltip.Provider>
          <Controls.Group className={cn('video-controls-primary', 'video-controls-spaced')}>
            <ButtonTooltip side="top">
              <PlayButton />
            </ButtonTooltip>
            <LiveButton />
            <div aria-hidden="true" className={'video-controls-spacer'} />
            <VolumePopover />
            <CaptionsMenu className={'video-controls-captions-menu'} />
            <ButtonTooltip side="top">
              <CastButton />
            </ButtonTooltip>
            <ButtonTooltip side="top">
              <AirPlayButton />
            </ButtonTooltip>
            {customControls}
            <ButtonTooltip side="top">
              <PiPButton />
            </ButtonTooltip>
            <ButtonTooltip side="top">
              <FullscreenButton />
            </ButtonTooltip>
          </Controls.Group>
        </Tooltip.Provider>
      </Controls.Content>
    </Controls.Root>
  );
}
