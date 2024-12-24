import { create } from "zustand";
import { persist } from "zustand/middleware";

interface SettingsState {
  videoLimit: number;
  chatPlaybackSmoothScroll: boolean;
  videoTheaterMode: boolean;
  showChatHistogram: boolean;
  setVideoLimit: (limit: number) => void;
  setChatPlaybackSmoothScroll: (smooth: boolean) => void;
  setVideoTheaterMode: (theaterMode: boolean) => void;
  setShowChatHistogram: (show: boolean) => void;
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

      setVideoLimit: (limit: number) => set({ videoLimit: limit }),

      setChatPlaybackSmoothScroll: (smooth: boolean) =>
        set({ chatPlaybackSmoothScroll: smooth }),

      setVideoTheaterMode: (theaterMode: boolean) =>
        set({ videoTheaterMode: theaterMode }),

      setShowChatHistogram: (show: boolean) => set({ showChatHistogram: show }),
    }),
    {
      name: "settings-storage",
      partialize: (state) => ({
        videoLimit: state.videoLimit,
        chatPlaybackSmoothScroll: state.chatPlaybackSmoothScroll,
        showChatHistogram: state.showChatHistogram,
      }),
    }
  )
);

export default useSettingsStore;
