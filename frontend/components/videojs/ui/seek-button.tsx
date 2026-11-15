'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/buttons.css';
import { SeekButton as SeekButtonPrimitive } from '@videojs/react';
import { SeekIcon as SeekIconPrimitive } from '@videojs/react/icons';

import { Button } from '@/components/videojs/ui/button';
import { cn, resolveClassName } from '@/lib/resolve-class-name';

export type SeekButtonProps = Omit<SeekButtonPrimitive.Props, 'children'>;

export function SeekButton({ className, seconds = 10, ...props }: SeekButtonProps = {}) {
  return (
    <SeekButtonPrimitive
      render={<Button />}
      className={(state) => cn('media-seek-button', resolveClassName(className, state))}
      seconds={seconds}
      {...props}
    >
      <div className={'media-seek-button-content'}>
        <SeekIconPrimitive className={cn('media-button-icon', seconds < 0 && 'media-seek-button-backward-icon')} />
        <span
          className={cn(
            'media-seek-button-label',
            seconds < 0 ? 'media-seek-button-backward-label' : 'media-seek-button-forward-label'
          )}
        >
          {Math.abs(seconds)}
        </span>
      </div>
    </SeekButtonPrimitive>
  );
}
