import {
  useMutation,
  UseMutationOptions,
  useQuery,
  useQueryClient,
  UseQueryOptions,
} from "@tanstack/react-query";
import { ApiResponse } from "@/app/hooks/useAxios";
import { AxiosInstance } from "axios";
import { Video } from "./useVideos";

export interface Playback {
  id: string;
  vod_id: string;
  user_id: string;
  time: number;
  status: PlaybackStatus;
  updated_at: Date;
  created_at: Date;
}

export interface NullResponse {
  null: null;
}

export enum PlaybackStatus {
  InProgress = "in_progress",
  Finished = "finished",
}

export interface LastPlaybackResponse {
  playback: Playback[];
  data: LastPlayback[];
}

interface LastPlayback {
  playback: Playback;
  vod: Video;
}

const fetchPlaybackForVideo = async (
  axiosPrivate: AxiosInstance,
  videoId: string
): Promise<Playback> => {
  const response = await axiosPrivate.get<ApiResponse<Playback>>(
    `/api/v1/playback/${videoId}`
  );
  return response.data.data;
};

const useFetchPlaybackForVideo = (
  axiosPrivate: AxiosInstance,
  videoId: string,
  options?: Omit<
    UseQueryOptions<Playback, Error, Playback, [string, string]>,
    "queryKey" | "queryFn"
  >
) => {
  return useQuery<Playback, Error, Playback, [string, string]>({
    queryKey: ["playback-data", videoId],
    queryFn: () => fetchPlaybackForVideo(axiosPrivate, videoId),
    ...options,
    refetchInterval: false,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    refetchIntervalInBackground: false,
  });
};

const startPlaybackForVideo = async (
  axiosPrivate: AxiosInstance,
  videoId: string
): Promise<ApiResponse<NullResponse>> => {
  const response = await axiosPrivate.post(
    `/api/v1/playback/start?video_id=${videoId}`
  );
  return response.data.data;
};

const useStartPlaybackForVideo = (
  axiosPrivate: AxiosInstance,
  videoId: string,
  options?: Omit<
    UseMutationOptions<
      ApiResponse<NullResponse>,
      Error,
      void,
      [string, string]
    >,
    "queryKey" | "queryFn"
  >
) => {
  return useMutation<ApiResponse<NullResponse>, Error, void, [string, string]>({
    mutationFn: () => startPlaybackForVideo(axiosPrivate, videoId),
    ...options,
  });
};

type UpdatePlaybackProgressVariables = {
  axiosPrivate: AxiosInstance;
  videoId: string;
  time: number;
};

// API function that takes individual parameters
const updatePlaybackProgressForVideo = async (
  axiosPrivate: AxiosInstance,
  videoId: string,
  time: number
): Promise<ApiResponse<NullResponse>> => {
  const response = await axiosPrivate.post(`/api/v1/playback/progress`, {
    vod_id: videoId,
    time,
  });
  return response.data.data;
};

// Custom hook that uses the mutation with proper typing
const useUpdatePlaybackProgressForVideo = () => {
  return useMutation<
    ApiResponse<NullResponse>,
    Error,
    UpdatePlaybackProgressVariables
  >({
    mutationFn: ({ axiosPrivate, videoId, time }) =>
      updatePlaybackProgressForVideo(axiosPrivate, videoId, time),
  });
};

type SetPlaybackStatusForVideoVariables = {
  axiosPrivate: AxiosInstance;
  videoId: string;
  status: PlaybackStatus;
};

// API function that takes individual parameters
const setPlaybackProgressForVideo = async (
  axiosPrivate: AxiosInstance,
  videoId: string,
  status: PlaybackStatus
): Promise<ApiResponse<NullResponse>> => {
  const response = await axiosPrivate.post(`/api/v1/playback/status`, {
    vod_id: videoId,
    status,
  });
  return response.data.data;
};

// Custom hook that uses the mutation with proper typing
const useSetPlaybackProgressForVideo = () => {
  return useMutation<
    ApiResponse<NullResponse>,
    Error,
    SetPlaybackStatusForVideoVariables
  >({
    mutationFn: ({ axiosPrivate, videoId, status }) =>
      setPlaybackProgressForVideo(axiosPrivate, videoId, status),
  });
};

const getLastPlaybackVideos = async (
  axiosPrivate: AxiosInstance,
  count: number
): Promise<LastPlaybackResponse> => {
  const response = await axiosPrivate.get<ApiResponse<LastPlaybackResponse>>(
    `/api/v1/playback/last`,
    {
      params: {
        limit: count,
      },
    }
  );
  return response.data.data;
};

const useGetLastPlaybackVideos = (
  axiosPrivate: AxiosInstance,
  count: number
) => {
  return useQuery({
    queryKey: ["playback-videos", count],
    queryFn: () => getLastPlaybackVideos(axiosPrivate, count),
  });
};

const markVideoAsWatched = async (
  axiosPrivate: AxiosInstance,
  videoId: string
): Promise<ApiResponse<NullResponse>> => {
  const response = await axiosPrivate.post(`/api/v1/playback/status`, {
    vod_id: videoId,
    status: PlaybackStatus.Finished,
  });
  return response.data;
};

export interface MarkVideoAsWatchedInput {
  axiosPrivate: AxiosInstance;
  videoId: string;
}

export interface DeletePlaybackInput {
  axiosPrivate: AxiosInstance;
  videoId: string;
}

const useMarkVideoAsWatched = () => {
  const queryClient = useQueryClient();
  return useMutation<ApiResponse<NullResponse>, Error, MarkVideoAsWatchedInput>(
    {
      mutationFn: ({ axiosPrivate, videoId }) =>
        markVideoAsWatched(axiosPrivate, videoId),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["playback-data"] });
      },
    }
  );
};

const deletePlayback = async (axiosPrivate: AxiosInstance, videoId: string) => {
  const response = await axiosPrivate.delete(`/api/v1/playback/${videoId}`);
  return response.data;
};

const useDeletePlayback = () => {
  const queryClient = useQueryClient();
  return useMutation<ApiResponse<NullResponse>, Error, DeletePlaybackInput>({
    mutationFn: ({ axiosPrivate, videoId }) =>
      deletePlayback(axiosPrivate, videoId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["playback-data"] });
    },
  });
};

export {
  useFetchPlaybackForVideo,
  fetchPlaybackForVideo,
  useStartPlaybackForVideo,
  useUpdatePlaybackProgressForVideo,
  useSetPlaybackProgressForVideo,
  useGetLastPlaybackVideos,
  useMarkVideoAsWatched,
  useDeletePlayback,
};
