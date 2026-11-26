'use client';

import { liveVideoFeatures, playbackRateFeature } from '@videojs/core/dom';
import { createPlayer } from '@videojs/react';

// Live archives support rewind, so add playback speed to the live preset.
export const { Player: GanymedeLivePlayer, usePlayer: useGanymedeLivePlayer } = createPlayer({
  features: [...liveVideoFeatures, playbackRateFeature],
  displayName: 'GanymedeLivePlayer',
});
