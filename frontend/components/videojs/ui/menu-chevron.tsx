import '../styles/base.css';
import '../styles/menus.css';
import { ChevronIcon as ChevronIconPrimitive } from '@videojs/react/icons';
import type { ClassValue } from 'cn';

import { cn } from '@/lib/utils';

export interface MenuChevronProps {
  back?: boolean;
  className?: ClassValue;
}

export function MenuChevron({ back = false, className }: MenuChevronProps = {}) {
  return (
    <ChevronIconPrimitive className={cn(back ? 'media-menu-back-chevron' : 'media-menu-forward-chevron', className)} />
  );
}
