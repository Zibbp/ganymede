import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import useAxios, { ApiResponse } from "./useAxios";
import { AxiosInstance } from "axios";
import { NullResponse } from "./usePlayback";
import { Video } from "./useVideos";

export interface Playlist {
  id: string;
  name: string;
  description: string;
  thumbnail_path: string;
  updated_at: string;
  created_at: string;
  edges: PlaylistEdges;
}

export interface PlaylistRule {
  id: string;
  name: string;
  field: PlaylistRuleField;
  operator: PlaylistRuleOperator;
  value: string;
  position: number;
  enabled: boolean;
}

export interface PlaylistRuleGroup {
  id: string;
  operator: "and" | "or";
  rules: Array<PlaylistRule>;
  position: number;
}

export enum PlaylistRuleField {
  Title = "title",
  Type = "type",
  Category = "category",
  Platform = "platform",
  ChannelName = "channel_name",
}

export enum PlaylistRuleOperator {
  Equals = "equals",
  Contains = "contains",
  Regex = "regex",
}

export enum PlaylistGroupOperator {
  AND = "AND",
  OR = "OR",
}

export interface PlaylistEdges {
  vods: Array<Video>;
  multistream_info: Array<PlaylistMultistreamInfo>;
}

export interface PlaylistMultistreamInfo {
  id: string;
  delay_ms: number;
  edges: PlaylistMultistreamInfoEdges;
}

export interface PlaylistMultistreamInfoEdges {
  vod: Video;
}

const getPlaylist = async (
  id: string,
  withMultistreamInfo: boolean
): Promise<Playlist> => {
  const response = await useAxios.get<ApiResponse<Playlist>>(
    `/api/v1/playlist/${id}`,
    {
      params: {
        with_multistream_info: withMultistreamInfo,
      },
    }
  );
  return response.data.data;
};

const useGetPlaylist = (id: string, withMultistreamInfo: boolean) => {
  return useQuery({
    queryKey: ["playlist", id, withMultistreamInfo],
    queryFn: () => getPlaylist(id, withMultistreamInfo),
    placeholderData: keepPreviousData,
  });
};

const getPlaylists = async (): Promise<Array<Playlist>> => {
  const response = await useAxios.get<ApiResponse<Array<Playlist>>>(
    `/api/v1/playlist`
  );
  return response.data.data;
};

const useGetPlaylists = () => {
  return useQuery({
    queryKey: ["playlists"],
    queryFn: () => getPlaylists(),
    placeholderData: keepPreviousData,
  });
};

const createPlaylist = async (
  axiosPrivate: AxiosInstance,
  name: string,
  description: string
): Promise<Playlist> => {
  const response = await axiosPrivate.post(`/api/v1/playlist`, {
    name,
    description,
  });
  return response.data.data;
};

interface CreatePlaylistVariables {
  axiosPrivate: AxiosInstance;
  name: string;
  description: string;
}

const useCreatePlaylist = () => {
  const queryClient = useQueryClient();
  return useMutation<Playlist, Error, CreatePlaylistVariables>({
    mutationFn: ({ axiosPrivate, name, description }) =>
      createPlaylist(axiosPrivate, name, description),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["playlists"] });
    },
  });
};

const editPlaylist = async (
  axiosPrivate: AxiosInstance,
  id: string,
  name: string,
  description: string
): Promise<Playlist> => {
  const response = await axiosPrivate.put(`/api/v1/playlist/${id}`, {
    name,
    description,
  });
  return response.data.data;
};

interface EditPlaylistVariables {
  axiosPrivate: AxiosInstance;
  id: string;
  name: string;
  description: string;
}

const useEditPlaylist = () => {
  const queryClient = useQueryClient();
  return useMutation<Playlist, Error, EditPlaylistVariables>({
    mutationFn: ({ axiosPrivate, id, name, description }) =>
      editPlaylist(axiosPrivate, id, name, description),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["playlists"] });
    },
  });
};

const deletePlaylist = async (
  axiosPrivate: AxiosInstance,
  id: string
): Promise<Playlist> => {
  const response = await axiosPrivate.delete(`/api/v1/playlist/${id}`);
  return response.data.data;
};

interface DeletePlaylistVariables {
  axiosPrivate: AxiosInstance;
  id: string;
}

const useDeletePlaylist = () => {
  const queryClient = useQueryClient();
  return useMutation<Playlist, Error, DeletePlaylistVariables>({
    mutationFn: ({ axiosPrivate, id }) => deletePlaylist(axiosPrivate, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["playlists"] });
    },
  });
};

type AddVideoToPlaylistVariables = {
  axiosPrivate: AxiosInstance;
  playlistId: string;
  videoId: string;
};

const addVideoToPlaylist = async (
  axiosPrivate: AxiosInstance,
  playlistId: string,
  videoId: string
): Promise<ApiResponse<NullResponse>> => {
  const response = await axiosPrivate.post(`/api/v1/playlist/${playlistId}`, {
    vod_id: videoId,
  });
  return response.data.data;
};

const useAddVideoToPlaylist = () => {
  return useMutation<
    ApiResponse<NullResponse>,
    Error,
    AddVideoToPlaylistVariables
  >({
    mutationFn: ({ axiosPrivate, playlistId, videoId }) =>
      addVideoToPlaylist(axiosPrivate, playlistId, videoId),
  });
};

type RemoveVideoFromPlaylist = {
  axiosPrivate: AxiosInstance;
  playlistId: string;
  videoId: string;
};

const removeVideoFromPlaylist = async (
  axiosPrivate: AxiosInstance,
  playlistId: string,
  videoId: string
): Promise<ApiResponse<NullResponse>> => {
  const response = await axiosPrivate.delete(
    `/api/v1/playlist/${playlistId}/vod`,
    {
      data: {
        vod_id: videoId,
      },
    }
  );
  return response.data.data;
};

const useRemoveVideoFromPlaylist = () => {
  return useMutation<ApiResponse<NullResponse>, Error, RemoveVideoFromPlaylist>(
    {
      mutationFn: ({ axiosPrivate, playlistId, videoId }) =>
        removeVideoFromPlaylist(axiosPrivate, playlistId, videoId),
    }
  );
};

const updateMultistreamVideoOffset = async (
  axiosPrivate: AxiosInstance,
  playlistId: string,
  videoId: string,
  delayMs: number
): Promise<NullResponse> => {
  const response = await axiosPrivate.put(
    `/api/v1/playlist/${playlistId}/multistream/delay`,
    {
      vod_id: videoId,
      delay_ms: delayMs,
    }
  );
  return response.data;
};

interface UpdateMultistreamVideoOffsetInput {
  axiosPrivate: AxiosInstance;
  playlistId: string;
  videoId: string;
  delayMs: number;
}

const useUpdateMultistreamVideoOffset = () => {
  return useMutation<NullResponse, Error, UpdateMultistreamVideoOffsetInput>({
    mutationFn: ({ axiosPrivate, playlistId, videoId, delayMs }) =>
      updateMultistreamVideoOffset(axiosPrivate, playlistId, videoId, delayMs),
  });
};

const getPlaylistRules = async (
  id: string
): Promise<Array<PlaylistRuleGroup>> => {
  const response = await useAxios.get<ApiResponse<Array<PlaylistRuleGroup>>>(
    `/api/v1/playlist/${id}/rules`
  );
  return response.data.data;
};

const useGetPlaylistRules = (id: string) => {
  return useQuery({
    queryKey: ["playlist_rules", id],
    queryFn: () => getPlaylistRules(id),
    placeholderData: keepPreviousData,
  });
};

const savePlaylistRules = async (
  id: string,
  rules: Array<PlaylistRuleGroup>,
  axiosPrivate: AxiosInstance
): Promise<ApiResponse<Array<PlaylistRuleGroup>>> => {
  const response = await axiosPrivate.put<
    ApiResponse<Array<PlaylistRuleGroup>>
  >(`/api/v1/playlist/${id}/rules`, {
    rule_groups: rules,
  });
  return response.data;
};

type savePlaylistVariables = {
  axiosPrivate: AxiosInstance;
  id: string;
  rules: Array<PlaylistRuleGroup>;
};

const useSavePlaylistRules = () => {
  const queryClient = useQueryClient();
  return useMutation<
    ApiResponse<Array<PlaylistRuleGroup>>,
    Error,
    savePlaylistVariables
  >({
    mutationFn: ({ axiosPrivate, id, rules }) =>
      savePlaylistRules(id, rules, axiosPrivate),
    onSuccess: (data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["playlist_rules", variables.id],
      });
    },
  });
};

type TestPlaylistRulesVariables = {
  axiosPrivate: AxiosInstance;
  playlistId: string;
  videoId: string;
};

const testPlaylistRules = async (
  axiosPrivate: AxiosInstance,
  playlistId: string,
  videoId: string
): Promise<ApiResponse<boolean>> => {
  const response = await axiosPrivate.post<ApiResponse<boolean>>(
    `/api/v1/playlist/${playlistId}/rules/test?video_id=${videoId}`
  );
  return response.data;
};

const useTestPlaylistRules = () => {
  return useMutation<ApiResponse<boolean>, Error, TestPlaylistRulesVariables>({
    mutationFn: ({ axiosPrivate, playlistId, videoId }) =>
      testPlaylistRules(axiosPrivate, playlistId, videoId),
  });
};

export {
  useGetPlaylists,
  useCreatePlaylist,
  useEditPlaylist,
  useDeletePlaylist,
  useGetPlaylist,
  useAddVideoToPlaylist,
  useRemoveVideoFromPlaylist,
  useUpdateMultistreamVideoOffset,
  useGetPlaylistRules,
  useSavePlaylistRules,
  useTestPlaylistRules,
};
