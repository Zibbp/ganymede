import { Controls, Time, Tooltip } from '@videojs/react';
import type { ComponentProps, ReactNode } from 'react';

import { AirPlayButton } from '@/components/videojs/ui/airplay-button';
import { ButtonTooltip } from '@/components/videojs/ui/button-tooltip';
import { CaptionsButton } from '@/components/videojs/ui/captions-button';
import { CastButton } from '@/components/videojs/ui/cast-button';
import { FullscreenButton } from '@/components/videojs/ui/fullscreen-button';
import { PiPButton } from '@/components/videojs/ui/pip-button';
import { PlayButton } from '@/components/videojs/ui/play-button';
import { TimeSlider } from '@/components/videojs/ui/time-slider';
import { VolumePopover } from '@/components/videojs/ui/volume-popover';
import { cn } from '@/lib/utils';

import { BookmarkMarkers, type TimelineBookmark } from '../bookmark-markers';
import { VideoSettingsMenu } from '../menus/settings-menu';

export interface DefaultVideoControlsProps {
  bookmarks?: readonly TimelineBookmark[];
  customControls?: ReactNode;
  renderThumbnail?: NonNullable<ComponentProps<typeof TimeSlider>>['renderThumbnail'];
}

export function DefaultVideoControls({ bookmarks = [], customControls, renderThumbnail }: DefaultVideoControlsProps = {}) {
  return (
    <Controls.Root>
      <Controls.Backdrop className={'video-controls-backdrop'} />
      <Controls.Content className={cn('video-controls', 'video-controls-content')}>
        <Tooltip.Provider>
          <Controls.Group className={'video-controls-primary'}>
            <ButtonTooltip side="top">
              <PlayButton />
            </ButtonTooltip>
            <VolumePopover className={'video-controls-volume-button'} />

            <Controls.Group className={'video-time-slider-group'}>
              <Time.Value className={cn('media-time-value', 'video-time-value')} type="current" />
              <div className="video-time-slider-with-bookmarks">
                <TimeSlider renderThumbnail={renderThumbnail} />
                <BookmarkMarkers bookmarks={bookmarks} />
              </div>
              <Time.Value className={cn('media-time-toggle', 'video-time-value')} type="remaining" toggle />
            </Controls.Group>

            <ButtonTooltip side="top">
              <CaptionsButton className={'video-controls-captions-button'} />
            </ButtonTooltip>
            <VideoSettingsMenu className={'video-controls-settings-button'} />
          </Controls.Group>

          <Controls.Group className={'video-controls-secondary'}>
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
