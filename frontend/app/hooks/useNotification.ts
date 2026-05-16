import { AxiosInstance } from "axios";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiResponse } from "./useAxios";

export enum NotificationType {
  Webhook = "webhook",
  Apprise = "apprise",
}

export enum AppriseType {
  Info = "info",
  Success = "success",
  Warning = "warning",
  Failure = "failure",
}

export enum AppriseFormat {
  Text = "text",
  HTML = "html",
  Markdown = "markdown",
}

export enum NotificationEventType {
  VideoSuccess = "video_success",
  LiveSuccess = "live_success",
  Error = "error",
  IsLive = "is_live",
}

export interface Notification {
  id: string;
  name: string;
  enabled: boolean;
  type: NotificationType;
  url: string;
  trigger_video_success: boolean;
  trigger_live_success: boolean;
  trigger_error: boolean;
  trigger_is_live: boolean;
  video_success_template: string;
  live_success_template: string;
  error_template: string;
  is_live_template: string;
  apprise_urls: string;
  apprise_title: string;
  apprise_type: AppriseType;
  apprise_tag: string;
  apprise_format: AppriseFormat;
  updated_at: string;
  created_at: string;
}

export type CreateNotificationInput = Omit<Notification, "id" | "updated_at" | "created_at">;
export type UpdateNotificationInput = Omit<Notification, "id" | "updated_at" | "created_at">;

// Get all notifications
const getNotifications = async (
  axiosPrivate: AxiosInstance
): Promise<Notification[]> => {
  const response = await axiosPrivate.get<ApiResponse<Notification[]>>(
    "/api/v1/notification"
  );
  return response.data.data;
};

const useGetNotifications = (axiosPrivate: AxiosInstance) => {
  return useQuery({
    queryKey: ["notifications"],
    queryFn: () => getNotifications(axiosPrivate),
  });
};

// Get single notification
const getNotification = async (
  axiosPrivate: AxiosInstance,
  id: string
): Promise<Notification> => {
  const response = await axiosPrivate.get<ApiResponse<Notification>>(
    `/api/v1/notification/${id}`
  );
  return response.data.data;
};

const useGetNotification = (axiosPrivate: AxiosInstance, id: string) => {
  return useQuery({
    queryKey: ["notification", id],
    queryFn: () => getNotification(axiosPrivate, id),
    enabled: !!id,
  });
};

// Create notification
const createNotification = async (
  axiosPrivate: AxiosInstance,
  input: CreateNotificationInput
): Promise<Notification> => {
  const response = await axiosPrivate.post<ApiResponse<Notification>>(
    "/api/v1/notification",
    input
  );
  return response.data.data;
};

interface CreateNotificationVariables {
  axiosPrivate: AxiosInstance;
  input: CreateNotificationInput;
}

const useCreateNotification = () => {
  const queryClient = useQueryClient();
  return useMutation<Notification, Error, CreateNotificationVariables>({
    mutationFn: ({ axiosPrivate, input }) =>
      createNotification(axiosPrivate, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });
};

// Update notification
const updateNotification = async (
  axiosPrivate: AxiosInstance,
  id: string,
  input: UpdateNotificationInput
): Promise<Notification> => {
  const response = await axiosPrivate.put<ApiResponse<Notification>>(
    `/api/v1/notification/${id}`,
    input
  );
  return response.data.data;
};

interface UpdateNotificationVariables {
  axiosPrivate: AxiosInstance;
  id: string;
  input: UpdateNotificationInput;
}

const useUpdateNotification = () => {
  const queryClient = useQueryClient();
  return useMutation<Notification, Error, UpdateNotificationVariables>({
    mutationFn: ({ axiosPrivate, id, input }) =>
      updateNotification(axiosPrivate, id, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });
};

// Delete notification
const deleteNotification = async (
  axiosPrivate: AxiosInstance,
  id: string
): Promise<void> => {
  await axiosPrivate.delete(`/api/v1/notification/${id}`);
};

interface DeleteNotificationVariables {
  axiosPrivate: AxiosInstance;
  id: string;
}

const useDeleteNotification = () => {
  const queryClient = useQueryClient();
  return useMutation<void, Error, DeleteNotificationVariables>({
    mutationFn: ({ axiosPrivate, id }) =>
      deleteNotification(axiosPrivate, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });
};

// Test notification
const testNotification = async (
  axiosPrivate: AxiosInstance,
  id: string,
  eventType: NotificationEventType
): Promise<void> => {
  await axiosPrivate.post(`/api/v1/notification/${id}/test`, {
    event_type: eventType,
  });
};

interface TestNotificationVariables {
  axiosPrivate: AxiosInstance;
  id: string;
  eventType: NotificationEventType;
}

const useTestNotification = () => {
  return useMutation<void, Error, TestNotificationVariables>({
    mutationFn: ({ axiosPrivate, id, eventType }) =>
      testNotification(axiosPrivate, id, eventType),
  });
};

export {
  useGetNotifications,
  useGetNotification,
  useCreateNotification,
  useUpdateNotification,
  useDeleteNotification,
  useTestNotification,
};
