import { create } from "zustand";
import { persist } from "zustand/middleware";

interface SettingsState {
  videoLimit: number;
  chatPlaybackSmoothScroll: boolean;
  videoTheaterMode: boolean;
  showChatHistogram: boolean;
  showProcessingVideosInRecentlyArchived: boolean;
  setVideoLimit: (limit: number) => void;
  setChatPlaybackSmoothScroll: (smooth: boolean) => void;
  setVideoTheaterMode: (theaterMode: boolean) => void;
  setShowChatHistogram: (show: boolean) => void;
  setShowProcessingVideosInRecentlyArchived: (show: boolean) => void;
}

// Create the store with persist middleware
const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      // Initial state
      videoLimit: 24,
      chatPlaybackSmoothScroll: false,
      videoTheaterMode: false,
      showChatHistogram: true,
      showProcessingVideosInRecentlyArchived: true,

      setVideoLimit: (limit: number) => set({ videoLimit: limit }),

      setChatPlaybackSmoothScroll: (smooth: boolean) =>
        set({ chatPlaybackSmoothScroll: smooth }),

      setVideoTheaterMode: (theaterMode: boolean) =>
        set({ videoTheaterMode: theaterMode }),

      setShowChatHistogram: (show: boolean) => set({ showChatHistogram: show }),

      setShowProcessingVideosInRecentlyArchived: (show: boolean) =>
        set({ showProcessingVideosInRecentlyArchived: show }),
    }),
    {
      name: "settings-storage",
      partialize: (state) => ({
        videoLimit: state.videoLimit,
        chatPlaybackSmoothScroll: state.chatPlaybackSmoothScroll,
        showChatHistogram: state.showChatHistogram,
        showProcessingVideosInRecentlyArchived:
          state.showProcessingVideosInRecentlyArchived,
      }),
    }
  )
);

export default useSettingsStore;
