# Updating the local Video.js skin source

## Scope and ownership

This directory contains editable Video.js 10 skin source installed from the
Video.js Shadcn registry. It is application-owned source, not generated build
output. Package updates do not update these files automatically, and running
Shadcn with `--overwrite` can replace local changes.

Use Video.js 10 APIs from `@videojs/*`. Do not introduce Video.js 8 APIs such
as `videojs()`, `registerPlugin`, `data-setup`, or `class="video-js"`.

The relevant configuration and dependencies are:

- `frontend/components.json`: the `@videojs` React/CSS registry.
- `frontend/package.json`: `@videojs/react`, `@videojs/core`, and
  `@videojs/hlsjs-video` must remain on compatible versions.
- `frontend/components/videojs/video/`: editable VOD skin.
- `frontend/components/videojs/live-video/`: editable processing/live skin.
- `frontend/components/videojs/ui/` and `styles/`: shared source used by both
  skins.

## Ganymede customizations to preserve

An upstream refresh must retain or deliberately reimplement all of these:

1. Note-derived timeline bookmarks:
   - `frontend/app/util/videoNotes.ts` parses timestamp links from VOD notes.
   - `frontend/app/components/videos/Player.tsx` passes the parsed bookmarks
     into the VOD skin.
   - `video/bookmark-markers.tsx` defines the markers and seek behavior.
   - `video/layout/controls.tsx` renders the bookmark layer beside the
     `TimeSlider`.
   - `video/skin.css` contains `video-time-slider-with-bookmarks` and
     `video-bookmark-*` styles.
2. Application control slots:
   - `video/skin.tsx` and `video/layout/controls.tsx` pass and render the
     `customControls` React node.
   - `live-video/skin.tsx` and `live-video/layout/controls.tsx` do the same for
     processing/live playback.
3. Ganymede player integration:
   - `frontend/app/components/videos/Player.tsx` imports both local skins and
     passes theater mode, absolute time, and chat visibility controls through
     `customControls`.
   - `PlayerTheaterModeIcon.tsx`, `PlayerAbsoluteTimeIcon.tsx`, and
     `PlayerHideChatIcon.tsx` use the local `Button` and `ButtonTooltip`
     primitives. Keep their accessible labels and `aria-pressed` state.

Do not restore the entire pre-update directory after refreshing it. Doing so
would silently discard upstream fixes. Merge the small customizations above
onto the new source instead.

## Update procedure

Run commands from `frontend/` unless noted otherwise.

1. Start from a clean worktree or make a checkpoint commit. Never run the
   overwrite commands with uncommitted skin work that cannot be recovered.
2. Read the version-matched bundled documentation:
   - `node_modules/@videojs/react/docs/llms.txt`
   - `node_modules/@videojs/react/docs/guides/customize-skins.md`
3. Select one target Video.js version. Use the version-matched CLI to confirm
   the React, existing Next.js, npm, Shadcn, CSS, video/live-video, and HLS
   setup before changing files. Pass every option explicitly so the output
   reports `Defaulted options: none`. Set `VIDEOJS_VERSION` to the selected
   concrete version, then run:

   ```bash
   npx "@videojs/cli@$VIDEOJS_VERSION" agents init --method shadcn --framework react --project existing --preset video --skin default --media hls --extensions none --source-url demo --package-manager npm --template next --styling css
   npx "@videojs/cli@$VIDEOJS_VERSION" agents init --method shadcn --framework react --project existing --preset live-video --skin default --media hls --extensions none --source-url demo --package-manager npm --template next --styling css
   ```
4. Upgrade the direct Video.js dependencies together. Do not upgrade only one
   of `@videojs/react`, `@videojs/core`, or `@videojs/hlsjs-video`. Keep
   `package.json` and `package-lock.json` synchronized:

   ```bash
   npm install "@videojs/react@$VIDEOJS_VERSION" "@videojs/core@$VIDEOJS_VERSION" "@videojs/hlsjs-video@$VIDEOJS_VERSION"
   ```
5. Before installing source, inspect both registry entries and confirm their
   declared `@videojs/react` version matches the selected package version:

   ```bash
   npx shadcn@latest view @videojs/video
   npx shadcn@latest view @videojs/live-video
   ```

   Stop if the registry and installed package versions do not match.
6. Refresh both editable skins:

   ```bash
   npx shadcn@latest add @videojs/video --overwrite --yes
   npx shadcn@latest add @videojs/live-video --overwrite --yes
   ```

7. Review every changed file. Reapply the Ganymede customizations listed above
   onto the new upstream structure. Adapt them to upstream API or layout
   changes instead of assuming the old patch still fits.
8. Check `frontend/app/components/videos/Player.tsx` still imports:

   ```tsx
   import { VideoSkin } from '@/components/videojs/video/skin';
   import { LiveVideoSkin } from '@/components/videojs/live-video/skin';
   ```

   It must not switch these two player paths back to packaged skin imports.
9. Review dependency and source diffs together. Confirm no unrelated Shadcn
   components, global theme tokens, or application files were overwritten.

## Validation

Run at least:

```bash
npx tsc --noEmit
npm run build
git diff --check
```

Run focused ESLint on every changed player and skin TypeScript file. Also run
`npm run lint`, but distinguish new failures from the repository's known
baseline failures rather than claiming the full lint check passed.

Manually verify, when a usable archive is available:

- archived MP4 and HLS playback;
- processing/live HLS playback;
- desktop, narrow/mobile, theater, and fullscreen layouts;
- play, seek, volume, captions, quality, PiP, fullscreen, and keyboard input;
- chapter segmentation and thumbnail previews;
- bookmark positions, hover/focus labels, keyboard activation, and seeking;
- theater mode, absolute-time display, and hide/show-chat controls;
- control auto-hide and player/chat synchronization.

Report the old and new Video.js versions, which Shadcn presets were refreshed,
which customizations required manual merging, and every validation command
that passed or could not be run.
