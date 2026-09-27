'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/indicators.css';
import { StatusIndicator as StatusIndicatorPrimitive } from '@videojs/react';
import {
  CaptionsOnIcon as CaptionsOnIconPrimitive,
  CaptionsOffIcon as CaptionsOffIconPrimitive,
  FullscreenEnterIcon as FullscreenEnterIconPrimitive,
  FullscreenExitIcon as FullscreenExitIconPrimitive,
  PipEnterIcon as PipEnterIconPrimitive,
  PipExitIcon as PipExitIconPrimitive,
  PlayIcon as PlayIconPrimitive,
  PauseIcon as PauseIconPrimitive,
} from '@videojs/react/icons';

import { resolveClassName } from '@/lib/resolve-class-name';
import { cn } from '@/lib/utils';

const TOP_STATUS_ACTIONS = ['toggleSubtitles', 'toggleFullscreen', 'togglePictureInPicture'] as const;

const PLAYBACK_STATUS_ACTIONS = ['togglePaused'] as const;

export type StatusIndicatorProps = Omit<StatusIndicatorPrimitive.RootProps, 'children'>;

export function StatusIndicator({ className, ...props }: StatusIndicatorProps = {}) {
  return (
    <StatusIndicatorPrimitive.Root
      actions={TOP_STATUS_ACTIONS}
      className={(state) => cn('media-indicator', 'media-status-indicator', resolveClassName(className, state))}
      {...props}
    >
      <div className={cn('media-indicator-content', 'media-status-indicator-content')}>
        <CaptionsOnIconPrimitive className={'media-status-indicator-captions-on-icon'} />
        <CaptionsOffIconPrimitive className={'media-status-indicator-captions-off-icon'} />
        <FullscreenEnterIconPrimitive className={'media-status-indicator-fullscreen-enter-icon'} />
        <FullscreenExitIconPrimitive className={'media-status-indicator-fullscreen-exit-icon'} />
        <PipEnterIconPrimitive className={'media-status-indicator-pip-enter-icon'} />
        <PipExitIconPrimitive className={'media-status-indicator-pip-exit-icon'} />
        <StatusIndicatorPrimitive.Value className={'media-status-indicator-value'} />
      </div>
    </StatusIndicatorPrimitive.Root>
  );
}

export type PlaybackStatusIndicatorProps = Omit<StatusIndicatorPrimitive.RootProps, 'children'>;

export function PlaybackStatusIndicator({ className, ...props }: PlaybackStatusIndicatorProps = {}) {
  return (
    <StatusIndicatorPrimitive.Root
      actions={PLAYBACK_STATUS_ACTIONS}
      className={(state) => cn('media-playback-status-indicator', resolveClassName(className, state))}
      {...props}
    >
      <PlayIconPrimitive className={'media-playback-status-indicator-play-icon'} />
      <PauseIconPrimitive className={'media-playback-status-indicator-pause-icon'} />
    </StatusIndicatorPrimitive.Root>
  );
}
