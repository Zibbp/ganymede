'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/popups.css';
import type { VolumeSliderProps as CoreVolumeSliderProps } from '@videojs/core';
import { VolumePopover as VolumePopoverPrimitive } from '@videojs/react';
import type { ClassValue } from 'cn';

import { ButtonTooltip } from '@/components/videojs/ui/button-tooltip';
import { MuteButton } from '@/components/videojs/ui/mute-button';
import { VolumeSlider } from '@/components/videojs/ui/volume-slider';
import { cn } from '@/lib/utils';

export interface VolumePopoverProps extends Omit<VolumePopoverPrimitive.RootProps, 'children'> {
  className?: ClassValue;
  orientation?: CoreVolumeSliderProps['orientation'];
  showTooltip?: boolean;
}

export function VolumePopover({
  className,
  showTooltip = false,
  side = 'top',
  orientation = 'vertical',
  ...props
}: VolumePopoverProps = {}) {
  return (
    <VolumePopoverPrimitive.Root openOnHover delay={200} closeDelay={100} side={side} {...props}>
      <ButtonTooltip delay={0} disabled={!showTooltip} sticky side="top">
        <VolumePopoverPrimitive.Trigger render={<MuteButton className={cn(className)} />} />
      </ButtonTooltip>
      <VolumePopoverPrimitive.Popup
        className={cn(
          'media-popup',
          'media-popup-safe-area',
          'media-popup-transition',
          'media-popup-surface',
          'media-volume-popover'
        )}
      >
        <VolumeSlider orientation={orientation} />
      </VolumePopoverPrimitive.Popup>
    </VolumePopoverPrimitive.Root>
  );
}
