'use client';

import '../styles/video/base.css';
import './skin.css';
import type { ComponentProps, ReactNode } from 'react';

import { BufferingIndicator } from '@/components/videojs/ui/buffering-indicator';
import { Container } from '@/components/videojs/ui/container';
import { ErrorDialog } from '@/components/videojs/ui/error-dialog';
import { Poster } from '@/components/videojs/ui/poster';
import { Title } from '@/components/videojs/ui/title';
import { cn } from '@/lib/utils';

import { LiveVideoGestures } from './behaviors/gestures';
import { LiveVideoHotkeys } from './behaviors/hotkeys';
import { LiveVideoStatusIndicators } from './display/status-indicators';
import { DefaultLiveVideoControls } from './layout/controls';

export interface LiveVideoSkinProps extends Omit<NonNullable<ComponentProps<typeof Container>>, 'children'> {
  children?: ReactNode;
  customControls?: ReactNode;
  renderPoster?: NonNullable<ComponentProps<typeof Poster>>['renderImage'];
}

export function LiveVideoSkin({ children, className, customControls, renderPoster, ...props }: LiveVideoSkinProps = {}) {
  return (
    <Container className={cn('video-skin', className)} data-theme="default" data-preset="live-video" {...props}>
      {children}
      <Poster renderImage={renderPoster} />
      <BufferingIndicator />
      <ErrorDialog />
      <Title />
      <DefaultLiveVideoControls customControls={customControls} />
      <LiveVideoHotkeys />
      <LiveVideoGestures />
      <LiveVideoStatusIndicators />
    </Container>
  );
}
