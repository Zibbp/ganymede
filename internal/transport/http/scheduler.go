package http

type SchedulerService interface {
	StartAppScheduler()
	StartLiveScheduler()
	StartJwksScheduler()
	// StartWatchVideoScheduler()
	StartTwitchCategoriesScheduler()
	StartPruneVideoScheduler()
}
