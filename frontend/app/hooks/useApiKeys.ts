import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiResponse } from "./useAxios";
import { AxiosInstance } from "axios";
import { NullResponse } from "./usePlayback";

// Tiers form a hierarchy within a resource: admin > write > read.
export enum ApiKeyTier {
  Read = "read",
  Write = "write",
  Admin = "admin",
}

// Resources mirror utils.ApiKeyResource on the backend. The catalog is
// intentionally complete; resources whose routes are not yet migrated to
// per-key scopes are flagged in API_KEY_RESOURCE_META below so the create
// form can surface a "(reserved)" note.
export enum ApiKeyResource {
  Wildcard = "*",
  Vod = "vod",
  Playlist = "playlist",
  Queue = "queue",
  Channel = "channel",
  Archive = "archive",
  Live = "live",
  User = "user",
  Config = "config",
  Playback = "playback",
  Notification = "notification",
  Task = "task",
  Chapter = "chapter",
  Category = "category",
  BlockedVideo = "blocked_video",
}

// API_KEY_RESOURCE_META describes each resource for the create form. The
// `enforced` flag tells the UI whether granting a scope on this resource
// has any effect today — when false, the route group still uses
// session-only auth and an API key holding the scope authenticates
// nothing. The label is shown as the MultiSelect group heading.
export const API_KEY_RESOURCE_META: Record<
  ApiKeyResource,
  { label: string; enforced: boolean }
> = {
  [ApiKeyResource.Wildcard]: { label: "All resources (wildcard)", enforced: true },
  [ApiKeyResource.Vod]: { label: "VODs", enforced: true },
  [ApiKeyResource.Playlist]: { label: "Playlists", enforced: true },
  [ApiKeyResource.Queue]: { label: "Queue", enforced: true },
  [ApiKeyResource.Channel]: { label: "Channels", enforced: false },
  [ApiKeyResource.Archive]: { label: "Archive", enforced: false },
  [ApiKeyResource.Live]: { label: "Live", enforced: false },
  [ApiKeyResource.User]: { label: "Users", enforced: false },
  [ApiKeyResource.Config]: { label: "Config", enforced: false },
  [ApiKeyResource.Playback]: { label: "Playback", enforced: false },
  [ApiKeyResource.Notification]: { label: "Notifications", enforced: false },
  [ApiKeyResource.Task]: { label: "Tasks", enforced: false },
  [ApiKeyResource.Chapter]: { label: "Chapters", enforced: false },
  [ApiKeyResource.Category]: { label: "Categories", enforced: false },
  [ApiKeyResource.BlockedVideo]: { label: "Blocked Videos", enforced: false },
};

// An ApiKeyScope is the on-the-wire string form: "<resource>:<tier>".
export type ApiKeyScope = string;

// makeScope is the typed constructor; mirrors utils.MakeApiKeyScope on
// the backend. Used to populate the create form's catalog.
export const makeScope = (r: ApiKeyResource, t: ApiKeyTier): ApiKeyScope =>
  `${r}:${t}`;

// API_KEY_SCOPES_CATALOG is the flat list of every defined scope. Source
// of truth for the form's MultiSelect data and zod validation.
export const API_KEY_SCOPES_CATALOG: ApiKeyScope[] = (
  Object.values(ApiKeyResource) as ApiKeyResource[]
).flatMap((r) =>
  (Object.values(ApiKeyTier) as ApiKeyTier[]).map((t) => makeScope(r, t))
);

// ApiKey is the JSON shape returned by the backend's apiKeyDTO. The full
// secret is never on this struct — it is only present in the
// CreateApiKeyResponse below, exactly once at creation time.
export interface ApiKey {
  id: string;
  name: string;
  description: string;
  prefix: string;
  // Granted permissions, each formatted as "<resource>:<tier>".
  scopes: ApiKeyScope[];
  last_used_at: string | null;
  created_at: string;
}

export interface CreateApiKeyInput {
  name: string;
  description: string;
  scopes: ApiKeyScope[];
}

export interface CreateApiKeyResponse {
  api_key: ApiKey;
  // The full token to surface in the show-once modal. Format:
  // "gym_<12-hex-prefix>_<43-char-base64url-secret>".
  secret: string;
}

const getApiKeys = async (
  axiosPrivate: AxiosInstance
): Promise<Array<ApiKey>> => {
  const response = await axiosPrivate.get<ApiResponse<Array<ApiKey>>>(
    "/api/v1/admin/api-keys"
  );
  return response.data.data ?? [];
};

const useGetApiKeys = (axiosPrivate: AxiosInstance) => {
  return useQuery({
    queryKey: ["apiKeys"],
    queryFn: () => getApiKeys(axiosPrivate),
  });
};

const createApiKey = async (
  axiosPrivate: AxiosInstance,
  input: CreateApiKeyInput
): Promise<CreateApiKeyResponse> => {
  const response = await axiosPrivate.post<ApiResponse<CreateApiKeyResponse>>(
    "/api/v1/admin/api-keys",
    input
  );
  return response.data.data;
};

interface CreateApiKeyVariables {
  axiosPrivate: AxiosInstance;
  input: CreateApiKeyInput;
}

const useCreateApiKey = () => {
  const queryClient = useQueryClient();
  return useMutation<CreateApiKeyResponse, Error, CreateApiKeyVariables>({
    mutationFn: ({ axiosPrivate, input }) => createApiKey(axiosPrivate, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["apiKeys"] });
    },
  });
};

const deleteApiKey = async (axiosPrivate: AxiosInstance, id: string) => {
  const response = await axiosPrivate.delete(`/api/v1/admin/api-keys/${id}`);
  return response.data;
};

interface DeleteApiKeyVariables {
  axiosPrivate: AxiosInstance;
  id: string;
}

const useDeleteApiKey = () => {
  const queryClient = useQueryClient();
  return useMutation<NullResponse, Error, DeleteApiKeyVariables>({
    mutationFn: ({ axiosPrivate, id }) => deleteApiKey(axiosPrivate, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["apiKeys"] });
    },
  });
};

export { useGetApiKeys, useCreateApiKey, useDeleteApiKey };
