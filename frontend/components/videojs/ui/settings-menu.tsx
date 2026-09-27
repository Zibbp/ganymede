'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/buttons.css';
import '../styles/menus.css';
import '../styles/popups.css';
import { settingsText } from '@videojs/core/i18n/text/menu';
import { Menu } from '@videojs/react';
import { Text as TextPrimitive } from '@videojs/react';
import { GearIcon as GearIconPrimitive } from '@videojs/react/icons';
import type { ClassValue } from 'cn';

import { Button } from '@/components/videojs/ui/button';
import { ButtonTooltip } from '@/components/videojs/ui/button-tooltip';
import { resolveClassName } from '@/lib/resolve-class-name';
import { cn } from '@/lib/utils';

export interface SettingsMenuProps extends Omit<Menu.RootProps, 'children'> {
  className?: ClassValue;

  children?: Menu.ContentProps['children'];
}

export function SettingsMenu({ children, className, ...props }: SettingsMenuProps) {
  return (
    <Menu.Root side="top" align="center" {...props}>
      <ButtonTooltip label={<TextPrimitive token={settingsText.key}>{settingsText.text}</TextPrimitive>} side="top">
        <Menu.Trigger
          render={<Button />}
          className={(state) => cn('media-settings-menu-trigger', resolveClassName(className, state))}
        >
          <GearIconPrimitive className={cn('media-button-icon-base', 'media-settings-menu-trigger-icon')} />
          <TextPrimitive className={'media-settings-menu-trigger-label'} token={settingsText.key}>
            {settingsText.text}
          </TextPrimitive>
        </Menu.Trigger>
      </ButtonTooltip>
      <Menu.Popup
        keepMounted
        className={cn('media-popup', 'media-popup-surface', 'media-menu-popup', 'media-menu-resizable-popup')}
      >
        <Menu.Content className={'media-menu-content'}>{children}</Menu.Content>
      </Menu.Popup>
    </Menu.Root>
  );
}
