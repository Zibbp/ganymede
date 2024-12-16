import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import useAxios, { ApiResponse } from "./useAxios";
import { Channel } from "./useChannels";
import { Playlist } from "./usePlaylist";
import { AxiosInstance } from "axios";
import { NullResponse } from "./usePlayback";

export interface PaginationResponse<T> {
  offset: number;
  limit: number;
  total_count: number;
  pages: number;
  data: T;
}

export interface Video {
  id: string;
  ext_id: string;
  clip_ext_vod_id?: string;
  platform: Platform;
  type: VideoType;
  title: string;
  duration: number;
  clip_vod_offset?: number;
  views: number;
  resolution: string;
  thumbnail_path: string;
  web_thumbnail_path: string;
  video_path: string;
  chat_path?: string;
  live_chat_path?: string;
  live_chat_convert_path?: string;
  chat_video_path?: string;
  info_path: string;
  folder_name: string;
  file_name: string;
  tmp_video_download_path: string;
  tmp_video_convert_path: string;
  tmp_chat_download_path: string;
  tmp_live_chat_download_path: string;
  tmp_live_chat_convert_path: string;
  tmp_chat_render_path: string;
  processing: boolean;
  streamed_at: Date;
  updated_at: Date;
  created_at: Date;
  edges: VideoEdges;
  local_views?: number;
  locked: boolean;
  caption_path: string;
}

export interface VideoEdges {
  channel: Channel;
  muted_segments?: MutedSegment[];
  chapters?: Chapter[];
}

export enum Platform {
  Twitch = "twitch",
  Youtube = "youtube",
}

export enum VideoType {
  Archive = "archive",
  Clip = "clip",
  Live = "live",
  Highlight = "highlight",
  Upload = "upload",
}

export interface MutedSegment {
  id: string;
  start: number;
  end: number;
}

export interface Chapter {
  id: string;
  start: number;
  end: number;
  type?: string;
  title?: string;
}

export interface CreateVodRequest {
  id: string;
  channel_id: string;
  ext_id: string;
  platform: Platform;
  type: VideoType;
  title: string;
  duration: number;
  views: number;
  resolution?: string;
  processing: boolean;
  thumbnail_path?: string;
  web_thumbnail_path: string;
  video_path: string;
  chat_path?: string;
  chat_video_path?: string;
  info_path?: string;
  caption_path?: string;
  streamed_at: string;
  locked: boolean;
}

type FetchVideoOptions = {
  id: string;
  with_channel: boolean;
  with_muted_segments: boolean;
  with_chapters: boolean;
};

const fetchVideo = async (
  id: string,
  withChannel: boolean,
  withChapters: boolean,
  withMutedSegments: boolean
): Promise<Video> => {
  const response = await useAxios.get<ApiResponse<Video>>(`/api/v1/vod/${id}`, {
    params: {
      with_channel: withChannel,
      with_chapters: withChapters,
      with_muted_segments: withMutedSegments,
    },
  });
  return response.data.data;
};

type FetchVideosFilterOptions =
  | {
      channel_id: string;
      playlist_id?: never;
      types: Array<VideoType>;
      limit: number;
      offset: number;
    }
  | {
      channel_id?: never;
      playlist_id: string;
      types: Array<VideoType>;
      limit: number;
      offset: number;
    }
  | {
      limit: number;
      offset: number;
    };

const fetchVideosFilter = async (
  limit: number,
  offset: number,
  types?: Array<VideoType>,
  channel_id?: string,
  playlist_id?: string
): Promise<PaginationResponse<Array<Video>>> => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const queryParams: { [key: string]: any } = {};

  // Add query parameters conditionally based on whether they are provided
  if (channel_id) {
    queryParams.channel_id = channel_id;
  }
  if (playlist_id) {
    queryParams.playlist_id = playlist_id;
  }
  if (types && types.length > 0) {
    queryParams.types = types.join(",");
  }

  const response = await useAxios.get<
    ApiResponse<PaginationResponse<Array<Video>>>
  >("/api/v1/vod/paginate", {
    params: {
      limit,
      offset,
      ...queryParams, // Spread the conditional query parameters
    },
  });

  return response.data.data;
};

const useFetchVideosFilter = (params: FetchVideosFilterOptions) => {
  // @ts-expect-error fine
  const { limit, offset, types, channel_id, playlist_id } = params;

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let queryKey: any[];
  if (channel_id) {
    queryKey = ["channel_videos", channel_id, limit, offset, types];
  } else if (playlist_id) {
    queryKey = ["playlist_videos", playlist_id, limit, offset, types];
  } else {
    queryKey = ["videos", limit, offset, types]; // Fetch videos without channel_id or playlist_id
  }

  return useQuery<PaginationResponse<Array<Video>>, Error>({
    queryKey,
    queryFn: () =>
      fetchVideosFilter(limit, offset, types, channel_id, playlist_id),
    placeholderData: keepPreviousData, // previous data is kept until the new data is swapped in. This prevents flashing when changing pages, filtering, etc.
  });
};

const useFetchVideo = (params: FetchVideoOptions) => {
  const { id, with_channel, with_chapters, with_muted_segments } = params;
  return useQuery({
    queryKey: ["video", id, with_channel, with_chapters, with_muted_segments],
    queryFn: () =>
      fetchVideo(id, with_channel, with_chapters, with_muted_segments),
  });
};

const searchVideos = async (
  limit: number,
  offset: number,
  query: string,
  types?: Array<VideoType>
): Promise<PaginationResponse<Array<Video>>> => {
  const queryParams: { [key: string]: unknown } = {};
  if (types && types.length > 0) {
    queryParams.types = types.join(",");
  }
  const response = await useAxios.get<
    ApiResponse<PaginationResponse<Array<Video>>>
  >("/api/v1/vod/search", {
    params: {
      limit,
      offset,
      q: query,
      ...queryParams,
    },
  });

  return response.data.data;
};

interface SearchVideosOptions {
  types: Array<VideoType>;
  limit: number;
  offset: number;
  query: string;
}

const useSearchVideos = (params: SearchVideosOptions) => {
  const { limit, offset, types, query } = params;
  return useQuery<PaginationResponse<Array<Video>>, Error>({
    queryKey: ["search", limit, offset, types, query],
    queryFn: () => searchVideos(limit, offset, query, types),
    placeholderData: keepPreviousData, // previous data is kept until the new data is swapped in. This prevents flashing when changing pages, filtering, etc.
  });
};

const getPlaylistsForVideo = async (
  video_id: string
): Promise<Array<Playlist>> => {
  const response = await useAxios.get<ApiResponse<Array<Playlist>>>(
    `/api/v1/vod/${video_id}/playlist`
  );
  return response.data.data;
};

const useGetPlaylistsForVideo = (video_id: string) => {
  return useQuery({
    queryKey: ["video", "playlists", video_id],
    queryFn: () => getPlaylistsForVideo(video_id),
    placeholderData: keepPreviousData,
  });
};

const deleteVideo = async (
  axiosPrivate: AxiosInstance,
  id: string,
  deleteFiles: boolean
): Promise<NullResponse> => {
  const response = await axiosPrivate.delete(`/api/v1/vod/${id}`, {
    params: {
      delete_files: deleteFiles,
    },
  });
  return response.data.data;
};

interface DeleteVideoVariables {
  axiosPrivate: AxiosInstance;
  id: string;
  deleteFiles: boolean;
}

const useDeleteVideo = () => {
  const queryClient = useQueryClient();
  return useMutation<NullResponse, Error, DeleteVideoVariables>({
    mutationFn: ({ axiosPrivate, id, deleteFiles }) =>
      deleteVideo(axiosPrivate, id, deleteFiles),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["videos"] });
    },
  });
};

const getVideosNoPaginate = async (): Promise<Array<Video>> => {
  const response = await useAxios.get(`/api/v1/vod`);
  return response.data.data;
};

const useGetVideosNoPaginate = () => {
  return useQuery({
    queryKey: ["videos"],
    queryFn: () => getVideosNoPaginate(),
  });
};

const createVideo = async (
  axiosPrivate: AxiosInstance,
  videoData: CreateVodRequest
): Promise<ApiResponse<Video>> => {
  const response = await axiosPrivate.post("/api/v1/vod", videoData);
  return response.data.data;
};

const editVideo = async (
  axiosPrivate: AxiosInstance,
  videoData: CreateVodRequest,
  videoId: string
): Promise<ApiResponse<Video>> => {
  const response = await axiosPrivate.put(`/api/v1/vod/${videoId}`, videoData);
  return response.data.data;
};

const useCreateVideo = () => {
  const queryClient = useQueryClient();
  return useMutation<
    ApiResponse<Video>,
    Error,
    {
      axiosPrivate: AxiosInstance;
      videoData: CreateVodRequest;
    }
  >({
    mutationFn: ({ axiosPrivate, videoData }) =>
      createVideo(axiosPrivate, videoData),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["videos"] });
    },
  });
};

const useEditVideo = () => {
  const queryClient = useQueryClient();
  return useMutation<
    ApiResponse<Video>,
    Error,
    {
      axiosPrivate: AxiosInstance;
      videoData: CreateVodRequest;
      videoId: string;
    }
  >({
    mutationFn: ({ axiosPrivate, videoData, videoId }) =>
      editVideo(axiosPrivate, videoData, videoId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["videos"] });
    },
  });
};

const lockVideo = async (
  axiosPrivate: AxiosInstance,
  videoId: string,
  locked: boolean
) => {
  const response = await axiosPrivate.post(
    `/api/v1/vod/${videoId}/lock`,
    {},
    {
      params: {
        locked: locked,
      },
    }
  );
  return response.data;
};

const useLockVideo = () => {
  const queryClient = useQueryClient();
  return useMutation<
    ApiResponse<NullResponse>,
    Error,
    {
      axiosPrivate: AxiosInstance;
      videoId: string;
      locked: boolean;
    }
  >({
    mutationFn: ({ axiosPrivate, videoId, locked }) =>
      lockVideo(axiosPrivate, videoId, locked),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["video"] });
    },
  });
};

const generateStaticThumbnail = async (
  axiosPrivate: AxiosInstance,
  videoId: string
) => {
  const response = await axiosPrivate.post(
    `/api/v1/vod/${videoId}/generate-static-thumbnail`
  );
  return response.data;
};

const useGenerateStaticThumbnail = () => {
  return useMutation<
    ApiResponse<NullResponse>,
    Error,
    {
      axiosPrivate: AxiosInstance;
      videoId: string;
    }
  >({
    mutationFn: ({ axiosPrivate, videoId }) =>
      generateStaticThumbnail(axiosPrivate, videoId),
  });
};

const getVideoByExternalId = async (extId: string): Promise<Video> => {
  const response = await useAxios.get<ApiResponse<Video>>(
    `/api/v1/vod/external_id/${extId}`
  );
  return response.data.data;
};

const useGetVideoByExternalId = (extId?: string) => {
  return useQuery({
    queryKey: ["video", extId],
    queryFn: () => getVideoByExternalId(extId!),
    enabled: !!extId,
    refetchInterval: false,
    refetchOnMount: false,
    refetchIntervalInBackground: false,
    retry: false,
  });
};

const getVideoClips = async (id: string): Promise<Video[]> => {
  const response = await useAxios.get<ApiResponse<Array<Video>>>(
    `/api/v1/vod/${id}/clips`
  );
  return response.data.data;
};

const useGetVideoClips = (id: string) => {
  return useQuery({
    queryKey: ["video_clips", id],
    queryFn: () => getVideoClips(id),
  });
};

export {
  fetchVideosFilter,
  useFetchVideosFilter,
  fetchVideo,
  useFetchVideo,
  useGetPlaylistsForVideo,
  useDeleteVideo,
  useGetVideosNoPaginate,
  useCreateVideo,
  useEditVideo,
  useLockVideo,
  useGenerateStaticThumbnail,
  useSearchVideos,
  useGetVideoByExternalId,
  useGetVideoClips,
};
