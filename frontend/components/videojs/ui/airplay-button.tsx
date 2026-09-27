'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/buttons.css';
import { AirPlayButton as AirPlayButtonPrimitive } from '@videojs/react';
import {
  AirPlayEnterIcon as AirPlayEnterIconPrimitive,
  AirPlayExitIcon as AirPlayExitIconPrimitive,
} from '@videojs/react/icons';

import { Button } from '@/components/videojs/ui/button';
import { resolveClassName } from '@/lib/resolve-class-name';
import { cn } from '@/lib/utils';

export type AirPlayButtonProps = Omit<AirPlayButtonPrimitive.Props, 'children'>;

export function AirPlayButton({ className, ...props }: AirPlayButtonProps = {}) {
  return (
    <AirPlayButtonPrimitive
      render={<Button />}
      className={(state) => cn('media-airplay-button', resolveClassName(className, state))}
      {...props}
    >
      <AirPlayEnterIconPrimitive className={cn('media-button-icon', 'media-airplay-button-enter-icon')} />
      <AirPlayExitIconPrimitive className={cn('media-button-icon', 'media-airplay-button-exit-icon')} />
    </AirPlayButtonPrimitive>
  );
}
