'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/menus.css';
import { Menu } from '@videojs/react';
import { CheckIcon as CheckIconPrimitive } from '@videojs/react/icons';

import { cn, resolveClassName } from '@/lib/resolve-class-name';

export interface RadioItemProps extends Omit<Menu.RadioItemProps, 'children'> {
  children?: Menu.RadioItemProps['children'];
}

export function RadioItem({ children, className, ...props }: RadioItemProps) {
  return (
    <Menu.RadioItem className={(state) => cn('media-menu-radio-item', resolveClassName(className, state))} {...props}>
      {children}
      <Menu.ItemIndicator forceMount className={'media-menu-item-indicator'}>
        <CheckIconPrimitive className={'media-menu-radio-item-icon'} />
      </Menu.ItemIndicator>
    </Menu.RadioItem>
  );
}
