package http

type SchedulerService interface {
	StartLiveScheduler()
	StartJwksScheduler()
}
