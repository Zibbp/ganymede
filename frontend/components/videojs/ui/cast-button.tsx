'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/buttons.css';
import { CastButton as CastButtonPrimitive } from '@videojs/react';
import { CastEnterIcon as CastEnterIconPrimitive, CastExitIcon as CastExitIconPrimitive } from '@videojs/react/icons';

import { Button } from '@/components/videojs/ui/button';
import { resolveClassName } from '@/lib/resolve-class-name';
import { cn } from '@/lib/utils';

export type CastButtonProps = Omit<CastButtonPrimitive.Props, 'children'>;

export function CastButton({ className, ...props }: CastButtonProps = {}) {
  return (
    <CastButtonPrimitive
      render={<Button />}
      className={(state) => cn('media-cast-button', resolveClassName(className, state))}
      {...props}
    >
      <CastEnterIconPrimitive className={cn('media-button-icon', 'media-cast-button-enter-icon')} />
      <CastExitIconPrimitive className={cn('media-button-icon', 'media-cast-button-exit-icon')} />
    </CastButtonPrimitive>
  );
}
