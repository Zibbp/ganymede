'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/buttons.css';
import { LiveButton as LiveButtonPrimitive } from '@videojs/react';

import { Button } from '@/components/videojs/ui/button';
import { resolveClassName } from '@/lib/resolve-class-name';
import { cn } from '@/lib/utils';

export type LiveButtonProps = Omit<LiveButtonPrimitive.Props, 'children'>;

export function LiveButton({ className, ...props }: LiveButtonProps = {}) {
  return (
    <LiveButtonPrimitive
      render={<Button />}
      className={(state) => cn('media-live-button', resolveClassName(className, state))}
      {...props}
    />
  );
}
