import { QueueLogType } from "../hooks/useQueue";

const windowFeatures = "left=100,top=100,width=720,height=620";

// open a queue task's log in a new window
export const openQueueTaskLog = (queueId: string, logName: QueueLogType) => {
  window.open(
    `/queue/logs/${queueId}?log=${logName}`,
    "Queue Task Log",
    windowFeatures
  );
};
