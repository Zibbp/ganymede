import type { ComponentProps } from 'react';

import { StatusAnnouncer } from '@/components/videojs/ui/status-announcer';
import { PlaybackStatusIndicator, StatusIndicator } from '@/components/videojs/ui/status-indicator';
import { VolumeIndicator } from '@/components/videojs/ui/volume-indicator';
import { cn } from '@/lib/resolve-class-name';

export type LiveVideoStatusIndicatorsProps = Omit<ComponentProps<'div'>, 'children'>;

export function LiveVideoStatusIndicators({ className, ...props }: LiveVideoStatusIndicatorsProps = {}) {
  return (
    <>
      <StatusAnnouncer />
      <div className={cn('video-status-indicators', className)} {...props}>
        <VolumeIndicator />
        <StatusIndicator />
        <PlaybackStatusIndicator />
      </div>
    </>
  );
}
