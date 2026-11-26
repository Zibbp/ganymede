import { Controls, BufferingIndicator, Time, Tooltip } from '@videojs/react';
import type { ReactNode } from 'react';

import { AirPlayButton } from '@/components/videojs/ui/airplay-button';
import { ButtonTooltip } from '@/components/videojs/ui/button-tooltip';
import { CaptionsMenu } from '@/components/videojs/ui/captions-menu';
import { CastButton } from '@/components/videojs/ui/cast-button';
import { FullscreenButton } from '@/components/videojs/ui/fullscreen-button';
import { LiveButton } from '@/components/videojs/ui/live-button';
import { PiPButton } from '@/components/videojs/ui/pip-button';
import { PlayButton } from '@/components/videojs/ui/play-button';
import { SeekButton } from '@/components/videojs/ui/seek-button';
import { TimeSlider } from '@/components/videojs/ui/time-slider';
import { VolumePopover } from '@/components/videojs/ui/volume-popover';
import { cn } from '@/lib/resolve-class-name';

import { LiveVideoSettingsMenu } from '../menus/settings-menu';

export interface DefaultLiveVideoControlsProps {
  customControls?: ReactNode;
}

export function DefaultLiveVideoControls({ customControls }: DefaultLiveVideoControlsProps = {}) {
  return (
    <Controls.Root>
      <BufferingIndicator className={'video-controls-center-buffering'}>
        <Controls.Group className={'video-controls-center'}>
          {/* Center seek buttons for touch devices. */}
          <SeekButton className={cn('video-controls-center-button', 'video-controls-center-seek')} seconds={-10} />
          <PlayButton
            className={cn('video-controls-center-button', 'video-controls-center-play')}
            iconClassName={'video-controls-center-play-icon'}
          />
          <SeekButton className={cn('video-controls-center-button', 'video-controls-center-seek')} seconds={10} />
        </Controls.Group>
      </BufferingIndicator>
      <Controls.Backdrop className={'video-controls-backdrop'} />
      <Controls.Content className={cn('video-controls', 'video-controls-content', 'video-controls-spaced')}>
        <Tooltip.Provider>
          <Controls.Group
            className={cn('video-controls-primary', 'video-controls-compact-primary', 'video-controls-spaced')}
          >
            <ButtonTooltip side="top">
              <PlayButton />
            </ButtonTooltip>
            <LiveButton />
            {/* Live archives can be rewound, so show a timeline. */}
            <Controls.Group className={'video-time-slider-group'}>
              <Time.Value className={cn('media-time-value', 'video-controls-current-time-value')} type="current" />
              <TimeSlider className={'video-controls-time-slider'} />
            </Controls.Group>
            <VolumePopover />
            <CaptionsMenu className={'video-controls-captions-menu'} />
            {/* Quality, audio, and speed settings. */}
            <LiveVideoSettingsMenu className={'video-controls-settings-button'} />
          </Controls.Group>

          <Controls.Group className={cn('video-controls-secondary', 'video-controls-top-controls')}>
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
