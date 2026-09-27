'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/buttons.css';
import { FullscreenButton as FullscreenButtonPrimitive } from '@videojs/react';
import {
  FullscreenEnterIcon as FullscreenEnterIconPrimitive,
  FullscreenExitIcon as FullscreenExitIconPrimitive,
} from '@videojs/react/icons';

import { Button } from '@/components/videojs/ui/button';
import { resolveClassName } from '@/lib/resolve-class-name';
import { cn } from '@/lib/utils';

export type FullscreenButtonProps = Omit<FullscreenButtonPrimitive.Props, 'children'>;

export function FullscreenButton({ className, ...props }: FullscreenButtonProps = {}) {
  return (
    <FullscreenButtonPrimitive
      render={<Button />}
      className={(state) => cn('media-fullscreen-button', resolveClassName(className, state))}
      {...props}
    >
      <FullscreenEnterIconPrimitive className={cn('media-button-icon', 'media-fullscreen-button-enter-icon')} />
      <FullscreenExitIconPrimitive className={cn('media-button-icon', 'media-fullscreen-button-exit-icon')} />
    </FullscreenButtonPrimitive>
  );
}
