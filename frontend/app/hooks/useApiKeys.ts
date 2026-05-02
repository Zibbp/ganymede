import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiResponse } from "./useAxios";
import { AxiosInstance } from "axios";
import { NullResponse } from "./usePlayback";

// Scope mirrors utils.ApiKeyScope on the backend.
export enum ApiKeyScope {
  Read = "read",
  Write = "write",
  Admin = "admin",
}

// ApiKey is the JSON shape returned by the backend's apiKeyDTO. The full
// secret is never on this struct — it is only present in the
// CreateApiKeyResponse below, exactly once at creation time.
export interface ApiKey {
  id: string;
  name: string;
  description: string;
  prefix: string;
  scope: ApiKeyScope;
  last_used_at: string | null;
  created_at: string;
}

export interface CreateApiKeyInput {
  name: string;
  description: string;
  scope: ApiKeyScope;
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
