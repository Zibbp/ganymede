'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/menus.css';
import { qualityText } from '@videojs/core/i18n/text/menu';
import { Menu } from '@videojs/react';
import { Text as TextPrimitive } from '@videojs/react';
import { SwitchesIcon as SwitchesIconPrimitive } from '@videojs/react/icons';
import { QualityRadioGroup } from '@videojs/react/ui/quality-radio-group';

import { MenuChevron } from '@/components/videojs/ui/menu-chevron';
import { RadioItem } from '@/components/videojs/ui/radio-item';

export type QualityMenuProps = Omit<Menu.RootProps, 'children'>;

export function QualityMenu({ ...props }: QualityMenuProps = {}) {
  return (
    <Menu.Root {...props}>
      <QualityRadioGroup.Root>
        <Menu.Trigger className={'media-menu-trigger-item'}>
          <SwitchesIconPrimitive className={'media-menu-trigger-item-icon'} />
          <TextPrimitive token={qualityText.key}>{qualityText.text}</TextPrimitive>
          <span className={'media-menu-hint'}>
            <QualityRadioGroup.Value className={'media-menu-hint-label'} />
            <MenuChevron />
          </span>
        </Menu.Trigger>
        <Menu.Content className={'media-menu-content'}>
          <Menu.Item className={'media-menu-back-item'}>
            <MenuChevron back />
            <TextPrimitive token={qualityText.key}>{qualityText.text}</TextPrimitive>
          </Menu.Item>
          <Menu.Separator className={'media-menu-separator'} />
          <QualityRadioGroup.Options
            className={'media-menu-radio-group'}
            renderItem={(props, item) => (
              <RadioItem {...props}>
                <span>
                  <span>{item.label}</span>
                  {item.tier ? <sup className={'media-menu-tier'}>{item.tier}</sup> : null}
                </span>
                {item.badge ? <span className={'media-menu-badge'}>{item.badge}</span> : null}
              </RadioItem>
            )}
          ></QualityRadioGroup.Options>
        </Menu.Content>
      </QualityRadioGroup.Root>
    </Menu.Root>
  );
}
