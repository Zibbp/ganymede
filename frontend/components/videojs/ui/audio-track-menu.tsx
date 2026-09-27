'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/menus.css';
import { audioText } from '@videojs/core/i18n/text/menu';
import { Menu } from '@videojs/react';
import { Text as TextPrimitive } from '@videojs/react';
import { SpeechIcon as SpeechIconPrimitive } from '@videojs/react/icons';
import { AudioTrackRadioGroup } from '@videojs/react/ui/audio-track-radio-group';

import { MenuChevron } from '@/components/videojs/ui/menu-chevron';
import { RadioItem } from '@/components/videojs/ui/radio-item';

export type AudioTrackMenuProps = Omit<Menu.RootProps, 'children'>;

export function AudioTrackMenu({ ...props }: AudioTrackMenuProps = {}) {
  return (
    <Menu.Root {...props}>
      <AudioTrackRadioGroup.Root>
        <Menu.Trigger className={'media-menu-trigger-item'}>
          <SpeechIconPrimitive className={'media-menu-trigger-item-icon'} />
          <TextPrimitive token={audioText.key}>{audioText.text}</TextPrimitive>
          <span className={'media-menu-hint'}>
            <AudioTrackRadioGroup.Value className={'media-menu-hint-label'} />
            <MenuChevron />
          </span>
        </Menu.Trigger>
        <Menu.Content className={'media-menu-content'}>
          <Menu.Item className={'media-menu-back-item'}>
            <MenuChevron back />
            <TextPrimitive token={audioText.key}>{audioText.text}</TextPrimitive>
          </Menu.Item>
          <Menu.Separator className={'media-menu-separator'} />
          <AudioTrackRadioGroup.Options
            className={'media-menu-radio-group'}
            renderItem={(props, item) => (
              <RadioItem {...props}>
                <span>{item.label}</span>
              </RadioItem>
            )}
          ></AudioTrackRadioGroup.Options>
        </Menu.Content>
      </AudioTrackRadioGroup.Root>
    </Menu.Root>
  );
}
