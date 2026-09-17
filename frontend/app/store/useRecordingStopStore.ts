import { useEffect } from "react";
import { create } from "zustand";
import { ArchiveStatus } from "@/app/util/archiveStatus";

interface RecordingStopState {
  requested: Record<string, boolean>;
  requestStop: (videoId: string) => void;
  clearStop: (videoId: string) => void;
}

// UI feedback shared across video cards and details, intentionally not persisted.
const useRecordingStopStore = create<RecordingStopState>((set) => ({
  requested: {},
  requestStop: (videoId) => set((state) => ({
    requested: { ...state.requested, [videoId]: true },
  })),
  clearStop: (videoId) => set((state) => {
    if (!state.requested[videoId]) return state;
    const requested = { ...state.requested };
    delete requested[videoId];
    return { requested };
  }),
}));

export function useRecordingStopping(videoId: string, status: ArchiveStatus) {
  const requested = useRecordingStopStore((state) => !!state.requested[videoId]);

  useEffect(() => {
    if (requested && status !== ArchiveStatus.Running) {
      useRecordingStopStore.getState().clearStop(videoId);
    }
  }, [videoId, status, requested]);

  return requested && status === ArchiveStatus.Running;
}

export default useRecordingStopStore;
