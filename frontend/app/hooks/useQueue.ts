import { AxiosInstance } from "axios";
import { ApiResponse } from "./useAxios";
import {
  useMutation,
  useQuery,
  useQueryClient,
  UseQueryOptions,
} from "@tanstack/react-query";
import { NullResponse } from "./usePlayback";
import { Video } from "./useVideos";

export interface Queue {
  id: string;
  live_archive: boolean;
  on_hold: boolean;
  video_processing: boolean;
  chat_processing: boolean;
  processing: boolean;
  task_vod_create_folder: QueueTaskStatus;
  task_vod_download_thumbnail: QueueTaskStatus;
  task_vod_save_info: QueueTaskStatus;
  task_video_download: QueueTaskStatus;
  task_video_convert: QueueTaskStatus;
  task_video_move: QueueTaskStatus;
  task_vprocessingideo_move: QueueTaskStatus;
  task_chat_download: QueueTaskStatus;
  task_chat_convert: QueueTaskStatus;
  task_chat_render: QueueTaskStatus;
  task_chat_move: QueueTaskStatus;
  archive_chat: boolean;
  render_chat: boolean;
  updated_at: string;
  created_at: string;
  edges: QueueEdges;
}

export interface QueueEdges {
  vod: Video;
}

export enum QueueTaskStatus {
  Success = "success",
  Running = "running",
  Pending = "pending",
  Failed = "failed",
}

export enum QueueTask {
  TaskVodCreateFolder = "task_vod_create_folder",
  TaskVodDownloadThumbnail = "task_vod_download_thumbnail",
  TaskVodSaveInfo = "task_vod_save_info",
  TaskVideoDownload = "task_video_download",
  TaskVideoConvert = "task_video_convert",
  TaskVideoMove = "task_video_move",
  TaskChatDownload = "task_chat_download",
  TaskChatConvert = "task_chat_convert",
  TaskChatRender = "task_chat_render",
  TaskChatMove = "task_chat_move",
  TaskLiveChatDownload = "task_live_chat_download",
  TaskLiveVideoDownload = "task_live_video_download",
}

export enum QueueLogType {
  Video = "video",
  VideoConvert = "video-convert",
  Chat = "chat",
  ChatRender = "chat-render",
  ChatConvert = "chat-convert",
}

const getQueueItems = async (
  axiosPrivate: AxiosInstance,
  processingOnly: boolean
): Promise<Array<Queue>> => {
  const response = await axiosPrivate.get<ApiResponse<Array<Queue>>>(
    `/api/v1/queue`,
    {
      params: {
        processing: processingOnly,
      },
    }
  );
  return response.data.data;
};

const useGetQueueItems = (
  axiosPrivate: AxiosInstance,
  processingOnly: boolean
) => {
  return useQuery({
    queryKey: ["queue", processingOnly],
    queryFn: () => getQueueItems(axiosPrivate, processingOnly),
  });
};

const stopQueueItem = async (
  axiosPrivate: AxiosInstance,
  id: string
): Promise<NullResponse> => {
  const response = await axiosPrivate.post(`/api/v1/queue/${id}/stop`);
  return response.data.data;
};

interface StopQueueItemVariables {
  axiosPrivate: AxiosInstance;
  id: string;
}

const getQueueItem = async (
  axiosPrivate: AxiosInstance,
  id: string
): Promise<Queue> => {
  const response = await axiosPrivate.get<ApiResponse<Queue>>(
    `/api/v1/queue/${id}`
  );
  return response.data.data;
};

type QueryFnType = typeof getQueueItem extends (
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  ...args: any[]
) => Promise<infer T>
  ? T
  : never;

type UseGetQueueItemOptions = Omit<
  UseQueryOptions<QueryFnType, Error, QueryFnType, string[]>,
  "queryKey" | "queryFn"
>;

const useGetQueueItem = (
  axiosPrivate: AxiosInstance,
  id: string,
  options?: UseGetQueueItemOptions
) => {
  return useQuery({
    queryKey: ["queue", id],
    queryFn: () => getQueueItem(axiosPrivate, id),
    ...options,
  });
};

const useStopQueueItem = () => {
  const queryClient = useQueryClient();
  return useMutation<NullResponse, Error, StopQueueItemVariables>({
    mutationFn: ({ axiosPrivate, id }) => stopQueueItem(axiosPrivate, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["queue"] });
    },
  });
};

const startQueueTask = async (
  axiosPrivate: AxiosInstance,
  queueId: string,
  taskName: QueueTask,
  continueWithSubsequent: boolean
): Promise<NullResponse> => {
  const response = await axiosPrivate.post(`/api/v1/queue/task/start`, {
    queue_id: queueId,
    task_name: taskName,
    continue: continueWithSubsequent,
  });
  return response.data.data;
};

interface StartQueueTaskVariables {
  axiosPrivate: AxiosInstance;
  queueId: string;
  taskName: QueueTask;
  continueWithSubsequent: boolean;
}

const useStartQueueTask = () => {
  return useMutation<NullResponse, Error, StartQueueTaskVariables>({
    mutationFn: ({ axiosPrivate, queueId, taskName, continueWithSubsequent }) =>
      startQueueTask(axiosPrivate, queueId, taskName, continueWithSubsequent),
  });
};

const editQueue = async (
  axiosPrivate: AxiosInstance,
  queue: Queue
): Promise<ApiResponse<NullResponse>> => {
  const response = await axiosPrivate.put(`/api/v1/queue/${queue.id}`, {
    id: queue.id,
    processing: queue.processing,
    on_hold: queue.on_hold,
    video_processing: queue.video_processing,
    chat_processing: queue.chat_processing,
    live_archive: queue.live_archive,
    task_vod_create_folder: queue.task_vod_create_folder,
    task_vod_download_thumbnail: queue.task_vod_download_thumbnail,
    task_vod_save_info: queue.task_vod_save_info,
    task_video_download: queue.task_video_download,
    task_video_convert: queue.task_video_convert,
    task_video_move: queue.task_video_move,
    task_chat_download: queue.task_chat_download,
    task_chat_convert: queue.task_chat_convert,
    task_chat_render: queue.task_chat_render,
    task_chat_move: queue.task_chat_move,
  });
  return response.data.data;
};

interface EditQueueVariables {
  axiosPrivate: AxiosInstance;
  queue: Queue;
}

const useEditQueue = () => {
  const queryClient = useQueryClient();
  return useMutation<ApiResponse<NullResponse>, Error, EditQueueVariables>({
    mutationFn: ({ axiosPrivate, queue }) => editQueue(axiosPrivate, queue),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["queue"] });
    },
  });
};

const deleteQueue = async (axiosPrivate: AxiosInstance, queueId: string) => {
  const response = await axiosPrivate.delete(`/api/v1/queue/${queueId}`);
  return response.data;
};

interface DeleteQueueVariables {
  axiosPrivate: AxiosInstance;
  queueId: string;
}

const useDeleteQueue = () => {
  const queryClient = useQueryClient();
  return useMutation<NullResponse, Error, DeleteQueueVariables>({
    mutationFn: ({ axiosPrivate, queueId }) =>
      deleteQueue(axiosPrivate, queueId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["queue"] });
    },
  });
};

const getQueueLogs = async (
  axiosPrivate: AxiosInstance,
  queueId: string,
  type: QueueLogType
): Promise<string> => {
  const response = await axiosPrivate.get<ApiResponse<string>>(
    `/api/v1/queue/${queueId}/tail`,
    {
      params: {
        type: type,
      },
    }
  );
  return response.data.data;
};

const useGetQueueLogs = (
  axiosPrivate: AxiosInstance,
  queueId: string,
  type: QueueLogType
) => {
  return useQuery({
    queryKey: ["queue_logs", queueId, type],
    queryFn: () => getQueueLogs(axiosPrivate, queueId, type),
    refetchInterval: 1000,
  });
};

export {
  useGetQueueItems,
  useStopQueueItem,
  useGetQueueItem,
  useStartQueueTask,
  useEditQueue,
  useDeleteQueue,
  useGetQueueLogs,
};
