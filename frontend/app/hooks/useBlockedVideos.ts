import { AxiosInstance } from "axios";
import { NullResponse } from "./usePlayback";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export interface BlockedVideo {
  id: string;
  created_at: Date;
}

const blockVideo = async (
  axiosPrivate: AxiosInstance,
  id: string
): Promise<NullResponse> => {
  const response = await axiosPrivate.post(`/api/v1/blocked-video/${id}`);
  return response.data.data;
};

interface BlockVideoVariables {
  axiosPrivate: AxiosInstance;
  id: string;
}

const useBlockVideo = () => {
  const queryClient = useQueryClient();
  return useMutation<NullResponse, Error, BlockVideoVariables>({
    mutationFn: ({ axiosPrivate, id }) => blockVideo(axiosPrivate, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["blocked_videos"] });
    },
  });
};

const getBlockedVideos = async (
  axiosPrivate: AxiosInstance
): Promise<Array<BlockedVideo>> => {
  const response = await axiosPrivate.get(`/api/v1/blocked-video`);
  return response.data.data;
};

const useGetBlockedVideos = (axiosPrivate: AxiosInstance) => {
  return useQuery({
    queryKey: ["blocked_videos"],
    queryFn: () => getBlockedVideos(axiosPrivate),
  });
};

const unblockVideo = async (axiosPrivate: AxiosInstance, videoId: string) => {
  const response = await axiosPrivate.delete(
    `/api/v1/blocked-video/${videoId}`
  );
  return response.data.data;
};

interface UnblockedVideoVariables {
  axiosPrivate: AxiosInstance;
  videoId: string;
}

const useUnblockVideo = () => {
  const queryClient = useQueryClient();
  return useMutation<NullResponse, Error, UnblockedVideoVariables>({
    mutationFn: ({ axiosPrivate, videoId }) =>
      unblockVideo(axiosPrivate, videoId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["blocked_videos"] });
    },
  });
};

export { useBlockVideo, useGetBlockedVideos, useUnblockVideo };
