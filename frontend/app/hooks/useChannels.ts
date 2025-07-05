import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import useAxios, { ApiResponse } from "./useAxios";
import { AxiosInstance } from "axios";

export interface Channel {
  id: string;
  ext_id: string;
  name: string;
  display_name: string;
  image_path: string;
  retention: boolean;
  retention_days: number;
  storage_size_bytes: number;
  updated_at: Date;
  created_at: Date;
}

const fetchChannels = async (): Promise<Array<Channel>> => {
  const response = await useAxios.get<ApiResponse<Array<Channel>>>(
    "/api/v1/channel"
  );
  return response.data.data;
};

const fetchChannelByName = async (name: string): Promise<Channel> => {
  const response = await useAxios.get<ApiResponse<Channel>>(
    `/api/v1/channel/name/${name}`
  );
  return response.data.data;
};

const useFetchChannels = () => {
  return useQuery({
    queryKey: ["channels"],
    queryFn: () => fetchChannels(),
  });
};

const useFetchChannelByName = (name: string) => {
  return useQuery({
    queryKey: ["channel", name],
    queryFn: () => fetchChannelByName(name),
  });
};

const createChannel = async (
  axiosPrivate: AxiosInstance,
  channel: Channel
): Promise<Channel> => {
  const response = await axiosPrivate.post(`/api/v1/channel`, {
    ext_id: channel.ext_id,
    name: channel.name,
    display_name: channel.display_name,
    image_path: channel.image_path,
    retention: channel.retention,
    retention_days: channel.retention_days,
  });
  return response.data.data;
};

interface CreateChannelVariables {
  axiosPrivate: AxiosInstance;
  channel: Channel;
}

const useCreateChannel = () => {
  const queryClient = useQueryClient();
  return useMutation<Channel, Error, CreateChannelVariables>({
    mutationFn: ({ axiosPrivate, channel }) =>
      createChannel(axiosPrivate, channel),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["channels"] });
    },
  });
};

const editChannel = async (
  axiosPrivate: AxiosInstance,
  channelId: string,
  channel: Channel
): Promise<Channel> => {
  const response = await axiosPrivate.put(`/api/v1/channel/${channelId}`, {
    name: channel.name,
    display_name: channel.display_name,
    image_path: channel.image_path,
    retention: channel.retention,
    retention_days: channel.retention_days,
  });
  return response.data.data;
};

interface EditChannelVariables {
  axiosPrivate: AxiosInstance;
  channelId: string;
  channel: Channel;
}

const useEditChannel = () => {
  const queryClient = useQueryClient();
  return useMutation<Channel, Error, EditChannelVariables>({
    mutationFn: ({ axiosPrivate, channelId, channel }) =>
      editChannel(axiosPrivate, channelId, channel),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["channels"] });
    },
  });
};

const updatedChannelImage = async (
  axiosPrivate: AxiosInstance,
  channelId: string
): Promise<Channel> => {
  const response = await axiosPrivate.post(
    `/api/v1/channel/${channelId}/update-image`
  );
  return response.data.data;
};

interface UpdateChannelImageVariables {
  axiosPrivate: AxiosInstance;
  channelId: string;
}

const useUpdateChannelImage = () => {
  return useMutation<Channel, Error, UpdateChannelImageVariables>({
    mutationFn: ({ axiosPrivate, channelId }) =>
      updatedChannelImage(axiosPrivate, channelId),
  });
};

const deleteChannel = async (
  axiosPrivate: AxiosInstance,
  channelId: string
) => {
  const response = await axiosPrivate.delete(`/api/v1/channel/${channelId}`);
  return response.data;
};

interface DeleteChannelVariables {
  axiosPrivate: AxiosInstance;
  channelId: string;
}

const useDeleteChannel = () => {
  const queryClient = useQueryClient();
  return useMutation<Channel, Error, DeleteChannelVariables>({
    mutationFn: ({ axiosPrivate, channelId }) =>
      deleteChannel(axiosPrivate, channelId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["channels"] });
    },
  });
};

export {
  useFetchChannels,
  fetchChannels,
  fetchChannelByName,
  useFetchChannelByName,
  useCreateChannel,
  useEditChannel,
  useUpdateChannelImage,
  useDeleteChannel,
};
