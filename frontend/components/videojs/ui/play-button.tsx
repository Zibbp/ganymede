'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/buttons.css';
import { PlayButton as PlayButtonPrimitive } from '@videojs/react';
import {
  RestartIcon as RestartIconPrimitive,
  PlayIcon as PlayIconPrimitive,
  PauseIcon as PauseIconPrimitive,
} from '@videojs/react/icons';
import type { ClassValue } from 'cn';

import { Button } from '@/components/videojs/ui/button';
import { cn, resolveClassName } from '@/lib/resolve-class-name';

export interface PlayButtonProps extends Omit<PlayButtonPrimitive.Props, 'children'> {
  iconClassName?: ClassValue;
}

export function PlayButton({ className, iconClassName, ...props }: PlayButtonProps = {}) {
  return (
    <PlayButtonPrimitive
      render={<Button />}
      className={(state) => cn('media-play-button', resolveClassName(className, state))}
      {...props}
    >
      <RestartIconPrimitive className={cn('media-button-icon', 'media-play-button-restart-icon', iconClassName)} />
      <PlayIconPrimitive className={cn('media-button-icon', 'media-play-button-play-icon', iconClassName)} />
      <PauseIconPrimitive className={cn('media-button-icon', 'media-play-button-pause-icon', iconClassName)} />
    </PlayButtonPrimitive>
  );
}
