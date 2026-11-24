package tasks_worker

import "testing"

func TestGetPeriodicTasksDisabledForTests(t *testing.T) {
	t.Setenv("TESTS_DISABLE_PERIODIC_JOBS", "true")

	tasks, err := getPeriodicTasks()
	if err != nil {
		t.Fatalf("getPeriodicTasks() error = %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("getPeriodicTasks() returned %d tasks, want 0", len(tasks))
	}
}
