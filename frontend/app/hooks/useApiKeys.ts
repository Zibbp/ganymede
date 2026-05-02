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

// Resources mirror utils.ApiKeyResource on the backend. Every resource
// here has at least one route that gates access on the matching scope,
// so granting any scope to a key produces a real, observable effect.
// Resources whose routes are intentionally public (chapter, category,
// twitch) or session-only (playback) are absent.
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
  Notification = "notification",
  Task = "task",
  BlockedVideo = "blocked_video",
  System = "system",
}

// API_KEY_RESOURCE_META labels each resource for the create form. Used
// as the MultiSelect group heading.
export const API_KEY_RESOURCE_META: Record<ApiKeyResource, { label: string }> = {
  [ApiKeyResource.Wildcard]: { label: "All resources (wildcard)" },
  [ApiKeyResource.Vod]: { label: "VODs" },
  [ApiKeyResource.Playlist]: { label: "Playlists" },
  [ApiKeyResource.Queue]: { label: "Queue" },
  [ApiKeyResource.Channel]: { label: "Channels" },
  [ApiKeyResource.Archive]: { label: "Archive" },
  [ApiKeyResource.Live]: { label: "Live" },
  [ApiKeyResource.User]: { label: "Users" },
  [ApiKeyResource.Config]: { label: "Config" },
  [ApiKeyResource.Notification]: { label: "Notifications" },
  [ApiKeyResource.Task]: { label: "Tasks" },
  [ApiKeyResource.BlockedVideo]: { label: "Blocked Videos" },
  [ApiKeyResource.System]: { label: "System (admin stats)" },
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
  // UUID of the admin who minted this key. Null for keys created
  // before the audit edge was added (Phase 3 follow-up).
  created_by_id: string | null;
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

// updateApiKey replaces the editable fields of an existing key. The
// server returns the updated apiKeyDTO; no secret is included (rotating
// the secret still means revoke + create).
const updateApiKey = async (
  axiosPrivate: AxiosInstance,
  id: string,
  input: CreateApiKeyInput
): Promise<ApiKey> => {
  const response = await axiosPrivate.put<ApiResponse<ApiKey>>(
    `/api/v1/admin/api-keys/${id}`,
    input
  );
  return response.data.data;
};

interface UpdateApiKeyVariables {
  axiosPrivate: AxiosInstance;
  id: string;
  input: CreateApiKeyInput;
}

const useUpdateApiKey = () => {
  const queryClient = useQueryClient();
  return useMutation<ApiKey, Error, UpdateApiKeyVariables>({
    mutationFn: ({ axiosPrivate, id, input }) =>
      updateApiKey(axiosPrivate, id, input),
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

export { useGetApiKeys, useCreateApiKey, useUpdateApiKey, useDeleteApiKey };
