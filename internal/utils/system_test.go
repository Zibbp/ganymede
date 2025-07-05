package utils

import (
	"testing"
)

func TestGetCPUCores(t *testing.T) {
	cores := GetCPUCores()
	if cores < 1 {
		t.Errorf("expected at least 1 CPU core, got %d", cores)
	}
}

func TestGetMemoryTotal(t *testing.T) {
	memTotal, err := GetMemoryTotal()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if memTotal <= 0 {
		t.Errorf("expected total memory > 0, got %d", memTotal)
	}
}
