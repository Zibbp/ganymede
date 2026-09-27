import type { ComponentProps } from 'react';

import { SeekIndicator } from '@/components/videojs/ui/seek-indicator';
import { StatusAnnouncer } from '@/components/videojs/ui/status-announcer';
import { PlaybackStatusIndicator, StatusIndicator } from '@/components/videojs/ui/status-indicator';
import { VolumeIndicator } from '@/components/videojs/ui/volume-indicator';
import { cn } from '@/lib/utils';

export type VideoStatusIndicatorsProps = Omit<ComponentProps<'div'>, 'children'>;

export function VideoStatusIndicators({ className, ...props }: VideoStatusIndicatorsProps = {}) {
  return (
    <>
      <StatusAnnouncer />
      <div className={cn('video-status-indicators', className)} {...props}>
        <VolumeIndicator />
        <StatusIndicator />
        <SeekIndicator />
        <PlaybackStatusIndicator />
      </div>
    </>
  );
}
