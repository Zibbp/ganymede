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

import { Button } from '@/components/videojs/ui/button';
import { resolveClassName } from '@/lib/resolve-class-name';
import { cn } from '@/lib/utils';

export type PlayButtonProps = Omit<PlayButtonPrimitive.Props, 'children'>;

export function PlayButton({ className, ...props }: PlayButtonProps = {}) {
  return (
    <PlayButtonPrimitive
      render={<Button />}
      className={(state) => cn('media-play-button', resolveClassName(className, state))}
      {...props}
    >
      <RestartIconPrimitive className={cn('media-button-icon', 'media-play-button-restart-icon')} />
      <PlayIconPrimitive className={cn('media-button-icon', 'media-play-button-play-icon')} />
      <PauseIconPrimitive className={cn('media-button-icon', 'media-play-button-pause-icon')} />
    </PlayButtonPrimitive>
  );
}
