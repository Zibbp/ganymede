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
}

export enum VideoQuality {
  Best = "best",
  quality1440p60 = "1440p60",
  quality1080p60 = "1080p60",
  quality720p60 = "720p60",
  quality480p = "480p30",
  quality360p = "360p30",
  quality160p = "160p30",
  audio = "audio",
}

const archiveVideo = async (
  axiosPrivate: AxiosInstance,
  video_id: string,
  channel_id: string,
  quality: VideoQuality,
  archive_chat: boolean,
  render_chat: boolean
): Promise<ApiResponse<NullResponse>> => {
  const response = await axiosPrivate.post(`/api/v1/archive/video`, {
    video_id,
    channel_id,
    quality,
    archive_chat,
    render_chat,
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
    }) =>
      archiveVideo(
        axiosPrivate,
        video_id,
        channel_id,
        quality,
        archive_chat,
        render_chat
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
