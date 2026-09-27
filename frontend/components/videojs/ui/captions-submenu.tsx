'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/menus.css';
import { captionsText } from '@videojs/core/i18n/text/menu';
import { Menu } from '@videojs/react';
import { Text as TextPrimitive } from '@videojs/react';
import { CaptionsOffIcon as CaptionsOffIconPrimitive } from '@videojs/react/icons';
import { CaptionsRadioGroup } from '@videojs/react/ui/captions-radio-group';

import { MenuChevron } from '@/components/videojs/ui/menu-chevron';
import { RadioItem } from '@/components/videojs/ui/radio-item';

export type CaptionsSubmenuProps = Omit<Menu.RootProps, 'children'>;

export function CaptionsSubmenu({ ...props }: CaptionsSubmenuProps = {}) {
  return (
    <Menu.Root {...props}>
      <CaptionsRadioGroup.Root>
        <Menu.Trigger className={'media-menu-trigger-item'}>
          <CaptionsOffIconPrimitive className={'media-menu-trigger-item-icon'} />
          <TextPrimitive token={captionsText.key}>{captionsText.text}</TextPrimitive>
          <span className={'media-menu-hint'}>
            <CaptionsRadioGroup.Value className={'media-menu-hint-label'} />
            <MenuChevron />
          </span>
        </Menu.Trigger>
        <Menu.Content className={'media-menu-content'}>
          <Menu.Item className={'media-menu-back-item'}>
            <MenuChevron back />
            <TextPrimitive token={captionsText.key}>{captionsText.text}</TextPrimitive>
          </Menu.Item>
          <Menu.Separator className={'media-menu-separator'} />
          <CaptionsRadioGroup.Options
            className={'media-menu-radio-group'}
            renderItem={(props, item) => (
              <RadioItem {...props}>
                <span>{item.label}</span>
              </RadioItem>
            )}
          ></CaptionsRadioGroup.Options>
        </Menu.Content>
      </CaptionsRadioGroup.Root>
    </Menu.Root>
  );
}
