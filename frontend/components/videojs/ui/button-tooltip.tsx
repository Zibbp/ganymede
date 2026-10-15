'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/popups.css';
import { Tooltip } from '@videojs/react';
import type { ReactElement, ReactNode } from 'react';

import { cn } from '@/lib/resolve-class-name';

export interface ButtonTooltipProps extends Omit<Tooltip.RootProps, 'children'> {
  children: ReactElement;
  label?: ReactNode;
}

export function ButtonTooltip({ children, label, ...props }: ButtonTooltipProps) {
  return (
    <Tooltip.Root {...props}>
      <Tooltip.Trigger render={children} />
      <Tooltip.Popup
        className={cn(
          'media-popup',
          'media-popup-safe-area',
          'media-popup-transition',
          'media-popup-surface',
          'media-tooltip'
        )}
      >
        {label ?? <Tooltip.Label />}
        {!label && <Tooltip.Shortcut className={'media-tooltip-shortcut'} />}
      </Tooltip.Popup>
    </Tooltip.Root>
  );
}
