import { AxiosInstance } from "axios";
import { ApiResponse } from "./useAxios";
import { NullResponse } from "./usePlayback";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export interface ArchiveVideoInput {
  axiosPrivate: AxiosInstance;
  video_id: string;
  channel_id: string;
  quality: VideoQuality;
  archive_chat: boolean;
  render_chat: boolean;
  platform: Platforms;
}

export enum VideoQuality {
  Best = "best",
  quality1440p = "1440p",
  quality1080p = "1080p",
  quality720p = "720p",
  quality480p = "480p",
  quality360p = "360p",
  quality160p = "160p",
  audio = "audio",
}

export enum Platforms {
  Twitch = "twitch",
  Kick = "kick",
}

const archiveVideo = async (
  axiosPrivate: AxiosInstance,
  video_id: string,
  channel_id: string,
  quality: VideoQuality,
  archive_chat: boolean,
  render_chat: boolean,
  platform: Platforms
): Promise<ApiResponse<NullResponse>> => {
  const response = await axiosPrivate.post(`/api/v1/archive/video`, {
    video_id,
    channel_id,
    quality,
    archive_chat,
    render_chat,
    platform,
  });
  return response.data.data;
};

const useArchiveVideo = () => {
  const queryClient = useQueryClient();
  return useMutation<ApiResponse<NullResponse>, Error, ArchiveVideoInput>({
    mutationFn: ({
      axiosPrivate,
      video_id,
      channel_id,
      quality,
      archive_chat,
      render_chat,
      platform,
    }) =>
      archiveVideo(
        axiosPrivate,
        video_id,
        channel_id,
        quality,
        archive_chat,
        render_chat,
        platform
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["queue"] });
    },
  });
};

export interface ArchiveChannelInput {
  axiosPrivate: AxiosInstance;
  channel_name: string;
}

const archiveChannel = async (
  axiosPrivate: AxiosInstance,
  channel_name: string
): Promise<ApiResponse<NullResponse>> => {
  const response = await axiosPrivate.post(`/api/v1/archive/channel`, {
    channel_name,
  });
  return response.data.data;
};

const useArchiveChannel = () => {
  const queryClient = useQueryClient();
  return useMutation<ApiResponse<NullResponse>, Error, ArchiveChannelInput>({
    mutationFn: ({ axiosPrivate, channel_name }) =>
      archiveChannel(axiosPrivate, channel_name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["channels"] });
    },
  });
};

export { useArchiveVideo, useArchiveChannel };
