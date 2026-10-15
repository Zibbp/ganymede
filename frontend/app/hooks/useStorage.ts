import { AxiosInstance } from "axios";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiResponse } from "./useAxios";

export type StorageScanState = "idle" | "queued" | "running";

export interface StorageScanStatus {
  state: StorageScanState;
  started_at: string | null;
  last_completed_at: string | null;
  last_failed_at: string | null;
}

export interface StorageFinding {
  id: string;
  kind: string;
  path: string;
  relative_path: string;
  size_bytes: number;
  detected_at: string;
}

export interface StorageFindings {
  videos_directory: string;
  scan: StorageScanStatus;
  findings: StorageFinding[];
}

export interface StorageActionFailure {
  id: string;
  path: string;
  error: string;
}

export interface StorageActionResult {
  done: string[];
  failed: StorageActionFailure[];
}

const getStorageFindings = async (
  axiosPrivate: AxiosInstance
): Promise<StorageFindings> => {
  const response = await axiosPrivate.get<ApiResponse<StorageFindings>>(
    "/api/v1/admin/storage-findings"
  );
  return response.data.data;
};

// awaitingScan keeps the query polling after a scan was started, because a scan of a small
// library finishes between two polls and would otherwise never be seen at all.
const useGetStorageFindings = (axiosPrivate: AxiosInstance, awaitingScan = false) => {
  return useQuery({
    queryKey: ["storage-findings"],
    queryFn: () => getStorageFindings(axiosPrivate),
    refetchInterval: (query) =>
      awaitingScan || (query.state.data && query.state.data.scan.state !== "idle") ? 2000 : false,
  });
};

const deleteStorageFindings = async (
  axiosPrivate: AxiosInstance,
  ids: string[]
): Promise<StorageActionResult> => {
  const response = await axiosPrivate.post<ApiResponse<StorageActionResult>>(
    "/api/v1/admin/storage-findings/delete",
    { ids }
  );
  return response.data.data;
};

const importStorageFindings = async (
  axiosPrivate: AxiosInstance,
  ids: string[]
): Promise<StorageActionResult> => {
  const response = await axiosPrivate.post<ApiResponse<StorageActionResult>>(
    "/api/v1/admin/storage-findings/import",
    { ids }
  );
  return response.data.data;
};

interface StorageFindingsVariables {
  axiosPrivate: AxiosInstance;
  ids: string[];
}

const useDeleteStorageFindings = () => {
  const queryClient = useQueryClient();
  return useMutation<StorageActionResult, Error, StorageFindingsVariables>({
    mutationFn: ({ axiosPrivate, ids }) => deleteStorageFindings(axiosPrivate, ids),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["storage-findings"] });
    },
  });
};

const useImportStorageFindings = () => {
  const queryClient = useQueryClient();
  return useMutation<StorageActionResult, Error, StorageFindingsVariables>({
    mutationFn: ({ axiosPrivate, ids }) => importStorageFindings(axiosPrivate, ids),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["storage-findings"] });
      queryClient.invalidateQueries({ queryKey: ["videos"] });
    },
  });
};

export { useGetStorageFindings, useDeleteStorageFindings, useImportStorageFindings };
