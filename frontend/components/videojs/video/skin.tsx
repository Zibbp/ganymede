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

import { VideoGestures } from './behaviors/gestures';
import { VideoHotkeys } from './behaviors/hotkeys';
import { VideoStatusIndicators } from './display/status-indicators';
import { DefaultVideoControls } from './layout/controls';
import type { TimelineBookmark } from './bookmark-markers';

export interface VideoSkinProps extends Omit<NonNullable<ComponentProps<typeof Container>>, 'children'> {
  children?: ReactNode;
  bookmarks?: readonly TimelineBookmark[];
  customControls?: ReactNode;
  renderPoster?: NonNullable<ComponentProps<typeof Poster>>['renderImage'];
  renderThumbnail?: NonNullable<ComponentProps<typeof DefaultVideoControls>>['renderThumbnail'];
}

export function VideoSkin({
  children,
  bookmarks,
  className,
  customControls,
  renderPoster,
  renderThumbnail,
  ...props
}: VideoSkinProps = {}) {
  return (
    <Container className={cn('video-skin', className)} data-theme="default" data-preset="video" {...props}>
      {children}
      <Poster renderImage={renderPoster} />
      <BufferingIndicator />
      <ErrorDialog />
      <Title />

      <DefaultVideoControls
        bookmarks={bookmarks}
        customControls={customControls}
        renderThumbnail={renderThumbnail}
      />

      <VideoHotkeys />
      <VideoGestures />
      <VideoStatusIndicators />
    </Container>
  );
}
