import { AxiosInstance } from "axios";
import { ApiResponse } from "./useAxios";
import { useQuery } from "@tanstack/react-query";
import { Video } from "./useVideos";

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
  chat_downloader: string;
  streamlink: string;
}

export interface GanymedeVideoStatistics {
  video_count: number;
  channel_count: number;
  channel_videos: Record<string, number>;
  video_types: Record<string, number>;
}

export interface GanymedeSystemOverview {
  videos_directory_free_space: number; // Free space in bytes
  videos_directory_used_space: number; // Used space in bytes
  cpu_cores: number; // Number of CPU cores
  memory_total: number; // Total memory in bytes
}

export interface GanymedeStorageDistribution {
  storage_distribution: Record<string, number>; // Map of channel names to total storage used
  largest_videos: Video[]; // List of top largest videos
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
