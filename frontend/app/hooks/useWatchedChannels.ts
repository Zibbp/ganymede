import { AxiosInstance } from "axios";
import { Channel } from "./useChannels";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Category } from "./useCategory";
import { ApiResponse } from "./useAxios";
import { NullResponse } from "./usePlayback";
import { update } from "lodash";

export interface WatchedChannel {
  id: string;
  watch_live: boolean;
  watch_vod: boolean;
  download_archives: boolean;
  download_highlights: boolean;
  download_uploads: boolean;
  download_sub_only: boolean;
  is_live: boolean;
  archive_chat: boolean;
  resolution: string;
  last_live: string;
  render_chat: boolean;
  video_age: number;
  apply_categories_to_live: boolean;
  watch_clips: boolean;
  clips_limit: number;
  clips_interval_days: number;
  clips_ignore_last_checked: boolean;
  update_metadata_minutes: number;
  updated_at: string;
  created_at: string;
  edges: WatchedChannelEdges;
}

export interface WatchedChannelEdges {
  channel: Channel;
  categories: Category[];
  title_regex: WatchedChannelTitleRegex[];
}

export interface WatchedChannelTitleRegex {
  id: string;
  negative: boolean;
  regex: string;
  apply_to_videos: boolean;
}

const getWatchedChannels = async (
  axiosPrivate: AxiosInstance
): Promise<Array<WatchedChannel>> => {
  const response = await axiosPrivate.get(`/api/v1/live`);
  return response.data.data;
};

const useGetWatchedChannesl = (axiosPrivate: AxiosInstance) => {
  return useQuery({
    queryKey: ["watched_channels"],
    queryFn: () => getWatchedChannels(axiosPrivate),
  });
};

const editWatchedChannel = async (
  axiosPrivate: AxiosInstance,
  watchedChannel: WatchedChannel,
  categories: string[]
): Promise<ApiResponse<NullResponse>> => {
  const response = await axiosPrivate.put(`/api/v1/live/${watchedChannel.id}`, {
    resolution: watchedChannel.resolution,
    archive_chat: watchedChannel.archive_chat,
    watch_live: watchedChannel.watch_live,
    watch_vod: watchedChannel.watch_vod,
    download_archives: watchedChannel.download_archives,
    download_highlights: watchedChannel.download_highlights,
    download_uploads: watchedChannel.download_uploads,
    render_chat: watchedChannel.render_chat,
    download_sub_only: watchedChannel.download_sub_only,
    categories: categories,
    video_age: watchedChannel.video_age,
    regex: watchedChannel.edges.title_regex,
    apply_categories_to_live: watchedChannel.apply_categories_to_live,
    watch_clips: watchedChannel.watch_clips,
    clips_limit: watchedChannel.clips_limit,
    clips_interval_days: watchedChannel.clips_interval_days,
    clips_ignore_last_checked: watchedChannel.clips_ignore_last_checked,
    update_metadata_minutes: watchedChannel.update_metadata_minutes,
  });
  return response.data.data;
};

const createWatchedChannel = async (
  axiosPrivate: AxiosInstance,
  channelId: string,
  watchedChannel: WatchedChannel,
  categories: string[]
): Promise<ApiResponse<NullResponse>> => {
  const response = await axiosPrivate.post(`/api/v1/live`, {
    channel_id: channelId,
    resolution: watchedChannel.resolution,
    archive_chat: watchedChannel.archive_chat,
    watch_live: watchedChannel.watch_live,
    watch_vod: watchedChannel.watch_vod,
    download_archives: watchedChannel.download_archives,
    download_highlights: watchedChannel.download_highlights,
    download_uploads: watchedChannel.download_uploads,
    render_chat: watchedChannel.render_chat,
    download_sub_only: watchedChannel.download_sub_only,
    categories: categories,
    video_age: watchedChannel.video_age,
    regex: watchedChannel.edges.title_regex,
    apply_categories_to_live: watchedChannel.apply_categories_to_live,
    watch_clips: watchedChannel.watch_clips,
    clips_limit: watchedChannel.clips_limit,
    clips_interval_days: watchedChannel.clips_interval_days,
    clips_ignore_last_checked: watchedChannel.clips_ignore_last_checked,
    update_metadata_minutes: watchedChannel.update_metadata_minutes,
  });
  return response.data.data;
};

interface EditWatchedChannelVariables {
  axiosPrivate: AxiosInstance;
  watchedChannel: WatchedChannel;
  categories: string[];
}

const useEditWatchedChannel = () => {
  const queryClient = useQueryClient();
  return useMutation<
    ApiResponse<NullResponse>,
    Error,
    EditWatchedChannelVariables
  >({
    mutationFn: ({ axiosPrivate, watchedChannel, categories }) =>
      editWatchedChannel(axiosPrivate, watchedChannel, categories),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["watched_channels"] });
    },
  });
};

const deleteWatchedChannel = async (
  axiosPrivate: AxiosInstance,
  watchedChannelId: string
): Promise<ApiResponse<NullResponse>> => {
  const response = await axiosPrivate.delete(
    `/api/v1/live/${watchedChannelId}`
  );
  return response.data.data;
};

interface DeleteWatchedChannelVariables {
  axiosPrivate: AxiosInstance;
  watchedChannelId: string;
}

const useDeleteWatchedChannel = () => {
  const queryClient = useQueryClient();
  return useMutation<
    ApiResponse<NullResponse>,
    Error,
    DeleteWatchedChannelVariables
  >({
    mutationFn: ({ axiosPrivate, watchedChannelId }) =>
      deleteWatchedChannel(axiosPrivate, watchedChannelId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["watched_channels"] });
    },
  });
};

interface CreateWatchedChannelVariables {
  axiosPrivate: AxiosInstance;
  channelId: string;
  watchedChannel: WatchedChannel;
  categories: string[];
}

const useCreateWatchedChannel = () => {
  const queryClient = useQueryClient();
  return useMutation<
    ApiResponse<NullResponse>,
    Error,
    CreateWatchedChannelVariables
  >({
    mutationFn: ({ axiosPrivate, channelId, watchedChannel, categories }) =>
      createWatchedChannel(axiosPrivate, channelId, watchedChannel, categories),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["watched_channels"] });
    },
  });
};

export {
  useGetWatchedChannesl,
  useEditWatchedChannel,
  useCreateWatchedChannel,
  useDeleteWatchedChannel,
};
