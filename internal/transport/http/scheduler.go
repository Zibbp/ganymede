package http

type SchedulerService interface {
	StartAppScheduler()
	StartLiveScheduler()
	StartQueueItemScheduler()
}
