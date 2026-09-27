'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/sliders.css';
import { VolumeSlider as VolumeSliderPrimitive } from '@videojs/react';

import { SliderFill, SliderThumb, SliderTrack } from '@/components/videojs/ui/slider';
import { resolveClassName } from '@/lib/resolve-class-name';
import { cn } from '@/lib/utils';

export type VolumeSliderProps = Omit<VolumeSliderPrimitive.RootProps, 'children'>;

export function VolumeSlider({ className, ...props }: VolumeSliderProps = {}) {
  return (
    <VolumeSliderPrimitive.Root
      className={(state) => cn('media-slider', 'media-volume-slider', resolveClassName(className, state))}
      thumbAlignment="edge"
      {...props}
    >
      <VolumeSliderPrimitive.Track render={<SliderTrack />}>
        <VolumeSliderPrimitive.Fill render={<SliderFill />} />
      </VolumeSliderPrimitive.Track>
      <VolumeSliderPrimitive.Thumb render={<SliderThumb />} className={'media-volume-slider-thumb'} />
    </VolumeSliderPrimitive.Root>
  );
}
