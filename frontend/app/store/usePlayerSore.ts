import { create } from "zustand";

interface VideoPlayerState {
  time: number;
  isPlaying: boolean;
  isPaused: boolean;
  updatePlayerState: (state: {
    time?: number;
    isPlaying?: boolean;
    isPaused?: boolean;
  }) => void;
}

const useVideoPlayerStore = create<VideoPlayerState>((set) => ({
  time: 0,
  isPlaying: false,
  isPaused: true,
  updatePlayerState: (newState) =>
    set((currentState) => ({
      time: newState.time ?? currentState.time,
      isPlaying: newState.isPlaying ?? currentState.isPlaying,
      isPaused: newState.isPaused ?? currentState.isPaused,
    })),
}));

export default useVideoPlayerStore;
