'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/indicators.css';
import { BufferingIndicator as BufferingIndicatorPrimitive } from '@videojs/react';
import { SpinnerIcon as SpinnerIconPrimitive } from '@videojs/react/icons';

import { resolveClassName } from '@/lib/resolve-class-name';
import { cn } from '@/lib/utils';

export type BufferingIndicatorProps = Omit<BufferingIndicatorPrimitive.Props, 'children'>;

export function BufferingIndicator({ className, ...props }: BufferingIndicatorProps = {}) {
  return (
    <BufferingIndicatorPrimitive
      className={(state) => cn('media-buffering-indicator', resolveClassName(className, state))}
      {...props}
    >
      <SpinnerIconPrimitive className={'media-buffering-indicator-spinner-icon'} />
    </BufferingIndicatorPrimitive>
  );
}
