'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/poster.css';
import { Poster as PosterPrimitive } from '@videojs/react';

import { resolveClassName } from '@/lib/resolve-class-name';
import { cn } from '@/lib/utils';

export interface PosterProps extends Omit<PosterPrimitive.ImageProps, 'children' | 'render'> {
  /** Draws the poster image in place of the one the skin renders. */
  renderImage?: PosterPrimitive.ImageProps['render'];

  children?: PosterPrimitive.RootProps['children'];
}

export function Poster({ children, className, renderImage, ...props }: PosterProps = {}) {
  return (
    <PosterPrimitive.Root className={(state) => cn('media-poster', resolveClassName(className, state))}>
      <PosterPrimitive.Image render={renderImage} className={'media-poster-image'} {...props} />

      {children}
    </PosterPrimitive.Root>
  );
}
