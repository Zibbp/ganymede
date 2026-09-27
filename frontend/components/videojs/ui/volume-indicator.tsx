'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/indicators.css';
import { VolumeIndicator as VolumeIndicatorPrimitive } from '@videojs/react';
import {
  VolumeHighIcon as VolumeHighIconPrimitive,
  VolumeLowIcon as VolumeLowIconPrimitive,
  VolumeOffIcon as VolumeOffIconPrimitive,
} from '@videojs/react/icons';

import { resolveClassName } from '@/lib/resolve-class-name';
import { cn } from '@/lib/utils';

export type VolumeIndicatorProps = Omit<VolumeIndicatorPrimitive.RootProps, 'children'>;

export function VolumeIndicator({ className, ...props }: VolumeIndicatorProps = {}) {
  return (
    <VolumeIndicatorPrimitive.Root
      className={(state) => cn('media-indicator', 'media-volume-indicator', resolveClassName(className, state))}
      {...props}
    >
      <VolumeIndicatorPrimitive.Fill className={cn('media-indicator-content', 'media-volume-indicator-fill')}>
        <VolumeHighIconPrimitive className={'media-volume-indicator-high-icon'} />
        <VolumeLowIconPrimitive className={'media-volume-indicator-low-icon'} />
        <VolumeOffIconPrimitive className={'media-volume-indicator-off-icon'} />
        <VolumeIndicatorPrimitive.Value className={'media-volume-indicator-value'} />
      </VolumeIndicatorPrimitive.Fill>
    </VolumeIndicatorPrimitive.Root>
  );
}
