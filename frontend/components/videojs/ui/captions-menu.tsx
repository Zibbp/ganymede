'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import '../styles/menus.css';
import '../styles/popups.css';
import { Menu as MenuPrimitive } from '@videojs/react';
import { CaptionsRadioGroup } from '@videojs/react/ui/captions-radio-group';

import { ButtonTooltip } from '@/components/videojs/ui/button-tooltip';
import { CaptionsButton } from '@/components/videojs/ui/captions-button';
import { RadioItem } from '@/components/videojs/ui/radio-item';
import { cn } from '@/lib/utils';

export interface CaptionsMenuProps extends Omit<MenuPrimitive.RootProps, 'children'> {
  className?: MenuPrimitive.TriggerProps['className'];
}

export function CaptionsMenu({ className, ...props }: CaptionsMenuProps = {}) {
  return (
    <MenuPrimitive.Root side="top" align="center" boundary="viewport" {...props}>
      <CaptionsRadioGroup.Root>
        <ButtonTooltip side="top">
          <MenuPrimitive.Trigger render={<CaptionsButton />} className={className} />
        </ButtonTooltip>
        <MenuPrimitive.Popup className={cn('media-popup', 'media-popup-surface', 'media-menu-popup')}>
          <MenuPrimitive.Content className={'media-menu-content'}>
            <CaptionsRadioGroup.Options
              className={'media-menu-radio-group'}
              renderItem={(props, item) => (
                <RadioItem {...props}>
                  <span>{item.label}</span>
                </RadioItem>
              )}
            ></CaptionsRadioGroup.Options>
          </MenuPrimitive.Content>
        </MenuPrimitive.Popup>
      </CaptionsRadioGroup.Root>
    </MenuPrimitive.Root>
  );
}
