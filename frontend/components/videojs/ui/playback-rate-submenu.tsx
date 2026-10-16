'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/menus.css';
import { speedText } from '@videojs/core/i18n/text/menu';
import { Menu } from '@videojs/react';
import { Text as TextPrimitive } from '@videojs/react';
import { SpeedIcon as SpeedIconPrimitive } from '@videojs/react/icons';
import { PlaybackRateRadioGroup } from '@videojs/react/ui/playback-rate-radio-group';

import { MenuChevron } from '@/components/videojs/ui/menu-chevron';
import { RadioItem } from '@/components/videojs/ui/radio-item';

export type PlaybackRateSubmenuProps = Omit<Menu.RootProps, 'children'>;

export function PlaybackRateSubmenu({ ...props }: PlaybackRateSubmenuProps = {}) {
  return (
    <Menu.Root {...props}>
      <PlaybackRateRadioGroup.Root>
        <Menu.Trigger className={'media-menu-trigger-item'}>
          <SpeedIconPrimitive className={'media-menu-trigger-item-icon'} />
          <TextPrimitive token={speedText.key}>{speedText.text}</TextPrimitive>
          <span className={'media-menu-hint'}>
            <PlaybackRateRadioGroup.Value className={'media-menu-hint-label'} />
            <MenuChevron />
          </span>
        </Menu.Trigger>
        <Menu.Content className={'media-menu-content'}>
          <Menu.Item className={'media-menu-back-item'}>
            <MenuChevron back />
            <TextPrimitive token={speedText.key}>{speedText.text}</TextPrimitive>
          </Menu.Item>
          <Menu.Separator className={'media-menu-separator'} />
          <PlaybackRateRadioGroup.Options
            className={'media-menu-radio-group'}
            renderItem={(props, item) => (
              <RadioItem {...props}>
                <span>{item.label}</span>
              </RadioItem>
            )}
          ></PlaybackRateRadioGroup.Options>
        </Menu.Content>
      </PlaybackRateRadioGroup.Root>
    </Menu.Root>
  );
}
