export enum ArchiveStatus {
  Queued = "queued",
  Running = "running",
  Finalizing = "finalizing",
  Completed = "completed",
  Failed = "failed",
}

export const activeArchiveStatuses = [ArchiveStatus.Queued, ArchiveStatus.Running, ArchiveStatus.Finalizing];
export const isArchiveActive = (status: ArchiveStatus) => activeArchiveStatuses.includes(status);
