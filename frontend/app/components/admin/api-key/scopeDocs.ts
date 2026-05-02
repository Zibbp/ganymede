// Reference data for the API key documentation modal. The route lists
// here mirror the per-route mapping in
// internal/transport/http/handler.go (groupV1Routes). Update both when
// migrating a new route group to RequireRoleOrScope.
//
// Strings are intentionally English-only — they reference HTTP routes
// and methods that aren't meaningful to translate. The modal's chrome
// (button label, title, footer) IS translated; see frontend/messages.

import { ApiKeyResource, ApiKeyTier } from "@/app/hooks/useApiKeys";

export interface TierDocs {
  // One-line summary of what the tier grants. Inclusive of lower
  // tiers within the same resource (admin includes write, write
  // includes read).
  summary: string;
  // Specific routes gated by exactly this tier. Routes covered by a
  // lower tier are not repeated here.
  routes: string[];
}

export interface ResourceDocs {
  resource: ApiKeyResource;
  // Display label shown in the accordion heading.
  label: string;
  // One-line description of the resource as a whole.
  description: string;
  // Per-tier documentation. A tier is omitted (rather than included as
  // empty) when it grants nothing extra beyond the lower tier — for
  // example, /vod has only public reads and a single DELETE, so its
  // read tier grants nothing extra.
  tiers: Partial<Record<ApiKeyTier, TierDocs>>;
}

// SCOPE_DOCS is the ordered list rendered in the modal. Wildcard goes
// first, then resources in roughly the order they appear in the
// backend catalog so power-users can scan top-to-bottom.
export const SCOPE_DOCS: ResourceDocs[] = [
  {
    resource: ApiKeyResource.Wildcard,
    label: "Wildcard (*)",
    description:
      "Grants the matching tier across every resource. Use only for full-trust automation; prefer specific resources for least-privilege keys.",
    tiers: {
      [ApiKeyTier.Read]: {
        summary: "Read access on every resource.",
        routes: [],
      },
      [ApiKeyTier.Write]: {
        summary: "Write access on every resource (includes read).",
        routes: [],
      },
      [ApiKeyTier.Admin]: {
        summary:
          "Admin access on every resource (includes write + read). Effectively a superuser key — be careful where you store it.",
        routes: [],
      },
    },
  },
  {
    resource: ApiKeyResource.Vod,
    label: "VODs (vod)",
    description: "Video metadata, locking, thumbnail generation, deletion.",
    tiers: {
      [ApiKeyTier.Write]: {
        summary:
          "Create and update VODs, lock/unlock, generate thumbnails, run ffprobe.",
        routes: [
          "POST /vod",
          "PUT /vod/:id",
          "POST /vod/:id/lock",
          "POST /vod/:id/generate-static-thumbnail",
          "POST /vod/:id/generate-sprite-thumbnails",
          "POST /vod/:id/ffprobe",
        ],
      },
      [ApiKeyTier.Admin]: {
        summary:
          "Everything in write, plus permanently deleting VODs and their files.",
        routes: ["DELETE /vod/:id"],
      },
    },
  },
  {
    resource: ApiKeyResource.Playlist,
    label: "Playlists (playlist)",
    description: "Playlist CRUD, multistream config, auto-fill rules.",
    tiers: {
      [ApiKeyTier.Read]: {
        summary: "Inspect playlist auto-fill rules.",
        routes: ["GET /playlist/:id/rules"],
      },
      [ApiKeyTier.Write]: {
        summary:
          "Create, edit, delete playlists, manage VOD membership, configure multistream and auto-fill rules.",
        routes: [
          "POST /playlist",
          "POST /playlist/:id (add VOD)",
          "DELETE /playlist/:id/vod",
          "DELETE /playlist/:id",
          "PUT /playlist/:id",
          "PUT /playlist/:id/multistream/delay",
          "PUT /playlist/:id/rules",
          "POST /playlist/:id/rules/test",
        ],
      },
    },
  },
  {
    resource: ApiKeyResource.Queue,
    label: "Queue (queue)",
    description:
      "Background archive/transcode jobs. Inspect status, run individual tasks, manage entries.",
    tiers: {
      [ApiKeyTier.Read]: {
        summary: "List queue items, inspect a single item, tail its logs.",
        routes: ["GET /queue", "GET /queue/:id", "GET /queue/:id/tail"],
      },
      [ApiKeyTier.Write]: {
        summary: "Update queue item state and start individual tasks.",
        routes: ["PUT /queue/:id", "POST /queue/task/start"],
      },
      [ApiKeyTier.Admin]: {
        summary:
          "Manually create queue items, delete entries, force-stop running jobs.",
        routes: [
          "POST /queue",
          "DELETE /queue/:id",
          "POST /queue/:id/stop",
        ],
      },
    },
  },
  {
    resource: ApiKeyResource.Channel,
    label: "Channels (channel)",
    description: "Channel metadata and image management.",
    tiers: {
      [ApiKeyTier.Write]: {
        summary: "Create and update channels, refresh channel images.",
        routes: [
          "POST /channel",
          "PUT /channel/:id",
          "POST /channel/:id/update-image",
        ],
      },
      [ApiKeyTier.Admin]: {
        summary: "Everything in write, plus deleting channels.",
        routes: ["DELETE /channel/:id"],
      },
    },
  },
  {
    resource: ApiKeyResource.Archive,
    label: "Archive (archive)",
    description: "Trigger archive jobs for channels, videos, and chat.",
    tiers: {
      [ApiKeyTier.Write]: {
        summary: "Submit channel and video archive jobs.",
        routes: ["POST /archive/channel", "POST /archive/video"],
      },
      [ApiKeyTier.Admin]: {
        summary:
          "Everything in write, plus running the Twitch live-chat conversion job.",
        routes: ["POST /archive/convert-twitch-live-chat"],
      },
    },
  },
  {
    resource: ApiKeyResource.Live,
    label: "Live (live)",
    description: "Watched-channel configuration for live archival.",
    tiers: {
      [ApiKeyTier.Read]: {
        summary: "List watched channels, run a manual liveness check.",
        routes: ["GET /live", "GET /live/check"],
      },
      [ApiKeyTier.Write]: {
        summary: "Add, edit, and remove watched channels.",
        routes: ["POST /live", "PUT /live/:id", "DELETE /live/:id"],
      },
    },
  },
  {
    resource: ApiKeyResource.User,
    label: "Users (user)",
    description: "Application user accounts.",
    tiers: {
      [ApiKeyTier.Read]: {
        summary: "List users and inspect a single user.",
        routes: ["GET /user", "GET /user/:id"],
      },
      [ApiKeyTier.Write]: {
        summary: "Update a user's username/role.",
        routes: ["PUT /user/:id"],
      },
      [ApiKeyTier.Admin]: {
        summary: "Everything in write, plus deleting users.",
        routes: ["DELETE /user/:id"],
      },
    },
  },
  {
    resource: ApiKeyResource.Config,
    label: "Config (config)",
    description: "Server-wide configuration (storage templates, intervals, etc.).",
    tiers: {
      [ApiKeyTier.Read]: {
        summary: "Read the current config.",
        routes: ["GET /config"],
      },
      [ApiKeyTier.Write]: {
        summary: "Replace the config.",
        routes: ["PUT /config"],
      },
    },
  },
  {
    resource: ApiKeyResource.Notification,
    label: "Notifications (notification)",
    description: "Notification rule CRUD and test sends.",
    tiers: {
      [ApiKeyTier.Read]: {
        summary: "List notifications, inspect a single rule.",
        routes: ["GET /notification", "GET /notification/:id"],
      },
      [ApiKeyTier.Write]: {
        summary: "Create, update, and test notification rules.",
        routes: [
          "POST /notification",
          "PUT /notification/:id",
          "POST /notification/:id/test",
        ],
      },
      [ApiKeyTier.Admin]: {
        summary: "Everything in write, plus deleting notification rules.",
        routes: ["DELETE /notification/:id"],
      },
    },
  },
  {
    resource: ApiKeyResource.Task,
    label: "Tasks (task)",
    description:
      "Manually trigger admin-only background tasks (cleanup, re-indexing, etc.).",
    tiers: {
      [ApiKeyTier.Admin]: {
        summary: "Start any admin-only background task by name.",
        routes: ["POST /task/start"],
      },
    },
  },
  {
    resource: ApiKeyResource.BlockedVideo,
    label: "Blocked Videos (blocked_video)",
    description: "Block list for videos that should be skipped during archival.",
    tiers: {
      [ApiKeyTier.Write]: {
        summary: "Add and remove videos from the block list.",
        routes: [
          "POST /blocked-video/:id",
          "DELETE /blocked-video/:id",
        ],
      },
    },
  },
  {
    resource: ApiKeyResource.System,
    label: "System (system)",
    description: "Server-wide statistics and info endpoints.",
    tiers: {
      [ApiKeyTier.Read]: {
        summary:
          "Read video stats, system overview (CPU/memory/disk), storage distribution, and Ganymede build info.",
        routes: [
          "GET /admin/video-statistics",
          "GET /admin/system-overview",
          "GET /admin/storage-distribution",
          "GET /admin/info",
        ],
      },
    },
  },
];

// SESSION_ONLY_NOTES lists endpoint groups that are intentionally NOT
// reachable with API keys, so admins minting a key know what's out of
// scope. Rendered in a footer panel below the per-resource accordion.
export const SESSION_ONLY_NOTES: { area: string; reason: string }[] = [
  {
    area: "/auth/* (login, register, OAuth, change-password)",
    reason:
      "Authentication endpoints — bootstrapping a session is the prerequisite for everything else.",
  },
  {
    area: "/admin/api-keys/* (create / list / revoke keys)",
    reason:
      "Key management is session-only on purpose. A stolen key cannot mint or escalate other keys.",
  },
  {
    area: "/playback/* (per-user watch progress)",
    reason:
      "Per-user UX state attributed to the session cookie. Scripts that need playback should authenticate as a real user via /auth/login.",
  },
  {
    area: "/twitch/*, /chapter/*, /category/*",
    reason: "Public reads — no authentication needed in either direction.",
  },
];
