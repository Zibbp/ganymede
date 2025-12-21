import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiResponse } from "./useAxios";
import { AxiosInstance } from "axios";
import { User } from "./useAuthentication";
import { NullResponse } from "./usePlayback";

export interface Config {
  live_check_interval_seconds: number;
  video_check_interval_minutes: number;
  registration_enabled: boolean;
  parameters: {
    twitch_token: string;
    video_convert: string;
    chat_render: string;
    yt_dlp_video: string;
  };
  archive: {
    save_as_hls: boolean;
    generate_sprite_thumbnails: boolean;
  };
  notifications: Notification;
  storage_templates: StorageTemplate;
  livestream: {
    proxies: ProxyListItem[];
    proxy_enabled: boolean;
    proxy_whitelist: string[];
    watch_while_archiving: boolean;
  };
}

export interface Notification {
  video_success_webhook_url: string;
  video_success_template: string;
  video_success_enabled: boolean;
  live_success_webhook_url: string;
  live_success_template: string;
  live_success_enabled: boolean;
  error_webhook_url: string;
  error_template: string;
  error_enabled: boolean;
  is_live_webhook_url: string;
  is_live_template: string;
  is_live_enabled: boolean;
}

export interface StorageTemplate {
  folder_template: string;
  file_template: string;
}

export enum ProxyType {
  TwitchHLS = "twitch_hls",
  HTTP = "http",
}

export interface ProxyListItem {
  url: string;
  header: string;
  proxy_type: ProxyType;
}

const getConfig = async (axiosPrivate: AxiosInstance): Promise<Config> => {
  const response = await axiosPrivate.get<ApiResponse<Config>>(
    "/api/v1/config"
  );
  return response.data.data;
};

const useGetConfig = (axiosPrivate: AxiosInstance) => {
  return useQuery({
    queryKey: ["admin_config"],
    queryFn: () => getConfig(axiosPrivate),
  });
};

const editConfig = async (
  axiosPrivate: AxiosInstance,
  config: Config
): Promise<User> => {
  const response = await axiosPrivate.put(`/api/v1/config`, {
    ...config,
  });
  return response.data.data;
};

interface EditConfigVariables {
  axiosPrivate: AxiosInstance;
  config: Config;
}

const useEditConfig = () => {
  const queryClient = useQueryClient();
  return useMutation<User, Error, EditConfigVariables>({
    mutationFn: ({ axiosPrivate, config }) => editConfig(axiosPrivate, config),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin_config"] });
    },
  });
};

export enum NotificationType {
  VideoSuccess = "video_success",
  LiveSuccess = "live_success",
  Error = "error",
  IsLive = "is_live",
}

const testNotification = async (
  axiosPrivate: AxiosInstance,
  type: NotificationType
): Promise<NullResponse> => {
  const response = await axiosPrivate.post(
    `/api/v1/notification/test`,
    {},
    {
      params: {
        type: type,
      },
    }
  );
  return response.data.data;
};

interface TestNotificationVariables {
  axiosPrivate: AxiosInstance;
  type: NotificationType;
}

const useTestNotification = () => {
  return useMutation<NullResponse, Error, TestNotificationVariables>({
    mutationFn: ({ axiosPrivate, type }) =>
      testNotification(axiosPrivate, type),
  });
};

export { useEditConfig, useGetConfig, useTestNotification };
