'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/sliders.css';
import type { ComponentProps } from 'react';

import { cn } from '@/lib/utils';

/** Shared slider track. */
export type SliderTrackProps = ComponentProps<'div'>;

export function SliderTrack({ className, ...props }: SliderTrackProps) {
  return <div className={cn('media-slider-track', className)} {...props} />;
}

/** Shared slider fill. */
export type SliderFillProps = ComponentProps<'div'>;

export function SliderFill({ className, ...props }: SliderFillProps) {
  return <div className={cn('media-slider-fill', className)} {...props} />;
}

/** Shared slider buffer. */
export type SliderBufferProps = ComponentProps<'div'>;

export function SliderBuffer({ className, ...props }: SliderBufferProps) {
  return <div className={cn('media-slider-buffer', className)} {...props} />;
}

/** Shared slider thumb. */
export type SliderThumbProps = ComponentProps<'div'>;

export function SliderThumb({ className, ...props }: SliderThumbProps) {
  return <div className={cn('media-slider-thumb', className)} {...props} />;
}
