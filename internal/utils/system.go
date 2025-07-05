package utils

import (
	"fmt"
	"runtime"

	"github.com/shirou/gopsutil/v4/mem"
)

// GetCPUCores returns the number of CPU cores available on the system.
func GetCPUCores() int {
	return runtime.NumCPU()
}

// GetMemoryTotal returns the total memory available on the system in bytes.
func GetMemoryTotal() (int64, error) {
	// Use mem.VirtualMemory() to get total memory in bytes
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, fmt.Errorf("error getting total memory: %w", err)
	}
	return int64(v.Total), nil
}
