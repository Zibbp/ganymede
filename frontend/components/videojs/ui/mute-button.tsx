'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/buttons.css';
import { MuteButton as MuteButtonPrimitive } from '@videojs/react';
import {
  VolumeOffIcon as VolumeOffIconPrimitive,
  VolumeLowIcon as VolumeLowIconPrimitive,
  VolumeHighIcon as VolumeHighIconPrimitive,
} from '@videojs/react/icons';

import { Button } from '@/components/videojs/ui/button';
import { resolveClassName } from '@/lib/resolve-class-name';
import { cn } from '@/lib/utils';

export type MuteButtonProps = Omit<MuteButtonPrimitive.Props, 'children'>;

export function MuteButton({ className, ...props }: MuteButtonProps = {}) {
  return (
    <MuteButtonPrimitive
      render={<Button />}
      className={(state) => cn('media-mute-button', resolveClassName(className, state))}
      {...props}
    >
      <VolumeOffIconPrimitive className={cn('media-button-icon', 'media-mute-button-off-icon')} />
      <VolumeLowIconPrimitive className={cn('media-button-icon', 'media-mute-button-low-icon')} />
      <VolumeHighIconPrimitive className={cn('media-button-icon', 'media-mute-button-high-icon')} />
    </MuteButtonPrimitive>
  );
}
