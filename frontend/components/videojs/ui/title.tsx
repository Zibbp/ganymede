'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/display.css';
import { Title as TitlePrimitive } from '@videojs/react';

import { cn, resolveClassName } from '@/lib/resolve-class-name';

export type TitleProps = Omit<TitlePrimitive.Props, 'children'>;

export function Title({ className, ...props }: TitleProps = {}) {
  return <TitlePrimitive className={(state) => cn('media-title', resolveClassName(className, state))} {...props} />;
}
