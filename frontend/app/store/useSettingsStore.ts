import { create } from "zustand";
import { persist } from "zustand/middleware";

interface SettingsState {
  videoLimit: number;
  adminItemsPerPage: number;
  chatPlaybackSmoothScroll: boolean;
  videoTheaterMode: boolean;
  hideChat: boolean;
  showChatHistogram: boolean;
  showProcessingVideosInRecentlyArchived: boolean;
  showAbsoluteTime: boolean;
  showChatTimestamps: boolean;
  setVideoLimit: (limit: number) => void;
  setAdminItemsPerPage: (limit: number) => void;
  setChatPlaybackSmoothScroll: (smooth: boolean) => void;
  setVideoTheaterMode: (theaterMode: boolean) => void;
  setHideChat: (hide: boolean) => void;
  setShowChatHistogram: (show: boolean) => void;
  setShowProcessingVideosInRecentlyArchived: (show: boolean) => void;
  setShowAbsoluteTime: (show: boolean) => void;
  setShowChatTimestamps: (show: boolean) => void;
}

// Create the store with persist middleware
const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      // Initial state
      videoLimit: 24,
      adminItemsPerPage: 20,
      chatPlaybackSmoothScroll: false,
      videoTheaterMode: false,
      hideChat: false,
      showChatHistogram: true,
      showProcessingVideosInRecentlyArchived: true,
      showAbsoluteTime: false,
      showChatTimestamps: false,

      setVideoLimit: (limit: number) => set({ videoLimit: limit }),

      setAdminItemsPerPage: (limit: number) =>
        set({ adminItemsPerPage: limit }),

      setChatPlaybackSmoothScroll: (smooth: boolean) =>
        set({ chatPlaybackSmoothScroll: smooth }),

      setVideoTheaterMode: (theaterMode: boolean) =>
        set({ videoTheaterMode: theaterMode }),

      setHideChat: (hide: boolean) => set({ hideChat: hide }),

      setShowChatHistogram: (show: boolean) => set({ showChatHistogram: show }),

      setShowProcessingVideosInRecentlyArchived: (show: boolean) =>
        set({ showProcessingVideosInRecentlyArchived: show }),

      setShowAbsoluteTime: (show: boolean) => set({ showAbsoluteTime: show }),

      setShowChatTimestamps: (show: boolean) =>
        set({ showChatTimestamps: show }),
    }),
    {
      name: "settings-storage",
      partialize: (state) => ({
        videoLimit: state.videoLimit,
        adminItemsPerPage: state.adminItemsPerPage,
        chatPlaybackSmoothScroll: state.chatPlaybackSmoothScroll,
        showChatHistogram: state.showChatHistogram,
        showProcessingVideosInRecentlyArchived:
          state.showProcessingVideosInRecentlyArchived,
        hideChat: state.hideChat,
        showAbsoluteTime: state.showAbsoluteTime,
        showChatTimestamps: state.showChatTimestamps,
      }),
    },
  ),
);

export default useSettingsStore;
