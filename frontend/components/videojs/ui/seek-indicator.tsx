'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/indicators.css';
import { SeekIndicator as SeekIndicatorPrimitive } from '@videojs/react';
import { ChevronIcon as ChevronIconPrimitive } from '@videojs/react/icons';

import { resolveClassName } from '@/lib/resolve-class-name';
import { cn } from '@/lib/utils';

export type SeekIndicatorProps = Omit<SeekIndicatorPrimitive.RootProps, 'children'>;

export function SeekIndicator({ className, ...props }: SeekIndicatorProps = {}) {
  return (
    <SeekIndicatorPrimitive.Root
      className={(state) => cn('media-seek-indicator', resolveClassName(className, state))}
      {...props}
    >
      <ChevronIconPrimitive className={'media-seek-indicator-icon'} />
      <SeekIndicatorPrimitive.Value className={'media-seek-indicator-value'} />
    </SeekIndicatorPrimitive.Root>
  );
}
