import { useMutation } from "@tanstack/react-query";
import { AxiosInstance } from "axios";

export enum Task {
  CheckLive = "check_live",
  CheckVod = "check_vod",
  CheckClips = "check_clips",
  GetJWKS = "get_jwks",
  StorageMigration = "storage_migration",
  PruneVideo = "prune_videos",
  SaveChapters = "save_chapters",
  UpdateStreamVodIds = "update_stream_vod_ids",
  GenerateSpriteThumbnails = "generate_sprite_thumbnails",
  UpdateVideoStorageUsage = "update_video_storage_usage",
}

const startTask = async (
  axiosPrivate: AxiosInstance,
  task: Task
): Promise<null> => {
  await axiosPrivate.post(`/api/v1/task/start`, {
    task: task,
  });
  return null;
};

type StartTaskVariables = {
  axiosPrivate: AxiosInstance;
  task: Task;
};

const useStartTask = () => {
  return useMutation<null, Error, StartTaskVariables>({
    mutationFn: ({ axiosPrivate, task }) => startTask(axiosPrivate, task),
  });
};

export { useStartTask };
