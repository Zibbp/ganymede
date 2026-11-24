import { Tooltip } from "@mantine/core";
import type { CSSProperties, ReactNode } from "react";

export type ChannelPlatformName = "twitch" | "kick" | "youtube" | string | undefined;

// Brand glyphs from Simple Icons (CC0-1.0), 24x24 viewBox.
const PLATFORMS: Record<string, { label: string; background: string; color: string; path: string }> = {
  twitch: {
    label: "Twitch",
    background: "#9146FF",
    color: "#FFFFFF",
    path: "M11.571 4.714h1.715v5.143H11.57zm4.715 0H18v5.143h-1.714zM6 0L1.714 4.286v15.428h5.143V24l4.286-4.286h3.428L22.286 12V0zm14.571 11.143l-3.428 3.428h-3.429l-3 3v-3H6.857V1.714h13.714Z",
  },
  kick: {
    label: "Kick",
    background: "#53FC18",
    color: "#000000",
    path: "M1.333 0h8v5.333H12V2.667h2.667V0h8v8H20v2.667h-2.667v2.666H20V16h2.667v8h-8v-2.667H12v-2.666H9.333V24h-8Z",
  },
};

type IconProps = {
  platform: ChannelPlatformName;
  size: number;
};

/** Small round brand mark for a channel's platform; renders nothing for unknown platforms. */
export const PlatformIcon = ({ platform, size }: IconProps) => {
  const brand = PLATFORMS[platform || "twitch"];
  if (!brand) return null;
  return (
    <Tooltip label={brand.label} openDelay={300} withinPortal>
      <span
        role="img"
        aria-label={brand.label}
        style={{
          display: "inline-flex",
          alignItems: "center",
          justifyContent: "center",
          width: size,
          height: size,
          borderRadius: "50%",
          background: brand.background,
          boxShadow: "0 0 0 2px var(--mantine-color-body)",
          flexShrink: 0,
        }}
      >
        <svg viewBox="0 0 24 24" width={size * 0.58} height={size * 0.58} aria-hidden="true">
          <path d={brand.path} fill={brand.color} />
        </svg>
      </span>
    </Tooltip>
  );
};

type BadgeProps = {
  platform: ChannelPlatformName;
  /** Badge diameter in px. */
  size: number;
  /** Distance from the bottom-right corner; negative values overhang the image. */
  offset?: number;
  children: ReactNode;
  style?: CSSProperties;
};

/** Wraps an avatar/image and pins the platform mark to its bottom-right corner. */
const PlatformBadge = ({ platform, size, offset = 0, children, style }: BadgeProps) => (
  <div style={{ position: "relative", display: "inline-block", lineHeight: 0, ...style }}>
    {children}
    <span style={{ position: "absolute", right: offset, bottom: offset, lineHeight: 0, zIndex: 1 }}>
      <PlatformIcon platform={platform} size={size} />
    </span>
  </div>
);

export default PlatformBadge;
