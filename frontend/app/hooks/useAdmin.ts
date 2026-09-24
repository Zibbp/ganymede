import { AxiosInstance } from "axios";
import { ApiResponse } from "./useAxios";
import { useQuery } from "@tanstack/react-query";

export interface GanymedeInformation {
  commit_hash: string;
  tag: string;
  build_time: string;
  uptime: string;
  program_versions: GanymedeProgramVersions;
}

export interface GanymedeProgramVersions {
  ffmpeg: string;
  twitch_downloader: string;
  yt_dlp: string;
}

export interface GanymedeVideoStatistics {
  video_count: number;
  channel_count: number;
  channel_videos: Record<string, number>;
  video_types: Record<string, number>;
  total_duration_seconds: number;
  total_views: number;
  total_local_views: number;
  total_storage_bytes: number;
}

export interface GanymedeQueueOverview {
  total: number;
  processing: number;
  on_hold: number;
  live_archiving: number;
  failed: number;
}

export interface GanymedeSystemOverview {
  videos_directory_free_space: number; // Free space in bytes
  videos_directory_used_space: number; // Used space in bytes
  videos_directory_total_space: number; // Total space in bytes (free + used)
  cpu_cores: number; // Number of CPU cores
  memory_total: number; // Total memory in bytes
  queue: GanymedeQueueOverview;
  user_count: number;
}

export interface GanymedeLargestVideo {
  id: string;
  title: string;
  channel_name: string;
  storage_size_bytes: number;
  duration: number;
  streamed_at: string;
}

export interface GanymedeStorageDistribution {
  storage_distribution: Record<string, number>; // Map of channel names to total storage used
  storage_by_type: Record<string, number>; // Map of video types to total storage used in bytes
  largest_videos: GanymedeLargestVideo[]; // List of top largest videos
}

const getGanymedeInformation = async (
  axiosPrivate: AxiosInstance
): Promise<GanymedeInformation> => {
  const response = await axiosPrivate.get<ApiResponse<GanymedeInformation>>(
    "/api/v1/admin/info"
  );
  return response.data.data;
};

const useGetGanymedeInformation = (axiosPrivate: AxiosInstance) => {
  return useQuery({
    queryKey: ["ganymede-information"],
    queryFn: () => getGanymedeInformation(axiosPrivate),
  });
};

const getGanymedeVideoStatistics = async (
  axiosPrivate: AxiosInstance
): Promise<GanymedeVideoStatistics> => {
  const response = await axiosPrivate.get<ApiResponse<GanymedeVideoStatistics>>(
    "/api/v1/admin/video-statistics"
  );
  return response.data.data;
};

const useGetGanymedeVideoStatistics = (axiosPrivate: AxiosInstance) => {
  return useQuery({
    queryKey: ["ganymede-video-statistics"],
    queryFn: () => getGanymedeVideoStatistics(axiosPrivate),
  });
};

const getGanymedeSystemOverview = async (
  axiosPrivate: AxiosInstance
): Promise<GanymedeSystemOverview> => {
  const response = await axiosPrivate.get<ApiResponse<GanymedeSystemOverview>>(
    "/api/v1/admin/system-overview"
  );
  return response.data.data;
};

const useGetGanymedeSystemOverview = (axiosPrivate: AxiosInstance) => {
  return useQuery({
    queryKey: ["ganymede-system-overview"],
    queryFn: () => getGanymedeSystemOverview(axiosPrivate),
    refetchInterval: 30_000,
    retry: 1,
  });
};

const getGanymedeStorageDistribution = async (
  axiosPrivate: AxiosInstance
): Promise<GanymedeStorageDistribution> => {
  const response = await axiosPrivate.get<
    ApiResponse<GanymedeStorageDistribution>
  >("/api/v1/admin/storage-distribution");
  return response.data.data;
};

const useGetGanymedeStorageDistribution = (axiosPrivate: AxiosInstance) => {
  return useQuery({
    queryKey: ["ganymede-storage-distribution"],
    queryFn: () => getGanymedeStorageDistribution(axiosPrivate),
  });
};

export {
  useGetGanymedeInformation,
  useGetGanymedeVideoStatistics,
  useGetGanymedeSystemOverview,
  useGetGanymedeStorageDistribution,
};
