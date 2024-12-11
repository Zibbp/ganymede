interface Data {
  time: number;
  isPlaying: boolean;
  isPaused: boolean;
}

class VideoEventBus {
  data: Data;

  constructor() {
    this.data = {
      time: 0,
      isPlaying: false,
      isPaused: true,
    };
  }

  setData(data: Data) {
    this.data = data;
  }

  getData() {
    return this.data;
  }
}

const videoEventBusInstance = new VideoEventBus();
export default videoEventBusInstance;
