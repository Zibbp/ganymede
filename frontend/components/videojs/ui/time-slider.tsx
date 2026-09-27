'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/popups.css';
import '../styles/sliders.css';
import type { SliderPreviewOverflow } from '@videojs/core';
import { Slider as SliderPrimitive, TimeSlider as TimeSliderPrimitive } from '@videojs/react';
import { SpinnerIcon as SpinnerIconPrimitive } from '@videojs/react/icons';

import { SliderBuffer, SliderFill, SliderThumb, SliderTrack } from '@/components/videojs/ui/slider';
import { resolveClassName } from '@/lib/resolve-class-name';
import { cn } from '@/lib/utils';

export interface TimeSliderProps extends Omit<TimeSliderPrimitive.RootProps, 'children'> {
  previewOverflow?: SliderPreviewOverflow | undefined;
  /** Draws the thumbnail preview image in place of the one the skin renders. */
  renderThumbnail?: SliderPrimitive.Thumbnail.ImageProps['render'];
}

export function TimeSlider({
  className,
  previewOverflow = 'visible',
  renderThumbnail,
  ...props
}: TimeSliderProps = {}) {
  return (
    <TimeSliderPrimitive.Root
      className={(state) => cn('media-slider', 'media-time-slider', resolveClassName(className, state))}
      {...props}
    >
      <TimeSliderPrimitive.Chapters
        className={'media-time-slider-chapters'}
        renderChapter={(props) => (
          <div className={'media-time-slider-chapter'} {...props}>
            <TimeSliderPrimitive.Track render={<SliderTrack />} className={'media-time-slider-chapter-track'}>
              <TimeSliderPrimitive.Buffer render={<SliderBuffer />} className={'media-time-slider-chapter-layer'} />
              <TimeSliderPrimitive.Fill render={<SliderFill />} className={'media-time-slider-chapter-layer'} />
            </TimeSliderPrimitive.Track>
          </div>
        )}
      ></TimeSliderPrimitive.Chapters>
      <TimeSliderPrimitive.Thumb render={<SliderThumb />} className={'media-time-slider-thumb'} />
      <TimeSliderPrimitive.Preview className={'media-slider-preview'} overflow={previewOverflow}>
        <SliderPrimitive.Thumbnail.Root
          className={cn('media-slider-preview-content', 'media-popup-surface', 'media-slider-thumbnail')}
        >
          <SliderPrimitive.Thumbnail.Image render={renderThumbnail} className={'media-slider-thumbnail-image'} />

          <SpinnerIconPrimitive className={'media-slider-thumbnail-spinner-icon'} />
        </SliderPrimitive.Thumbnail.Root>
        <div className={cn('media-slider-preview-content', 'media-time-slider-preview-content')}>
          <TimeSliderPrimitive.ChapterTitle className={'media-time-slider-chapter-title'} />
          <TimeSliderPrimitive.Value className={'media-time-slider-value'} type="pointer" />
        </div>
      </TimeSliderPrimitive.Preview>
    </TimeSliderPrimitive.Root>
  );
}
