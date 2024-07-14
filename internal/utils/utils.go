package utils

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func SanitizeFileName(fileName string) string {
	// Use ReplaceAll instead of regex for easier updating
	// Replace the unwanted characters with _
	fileName = strings.ReplaceAll(fileName, "/", "_")
	fileName = strings.ReplaceAll(fileName, ":", "_")
	fileName = strings.ReplaceAll(fileName, "*", "_")
	fileName = strings.ReplaceAll(fileName, "?", "_")
	fileName = strings.ReplaceAll(fileName, "\"", "_")
	fileName = strings.ReplaceAll(fileName, "<", "_")
	fileName = strings.ReplaceAll(fileName, ">", "_")
	fileName = strings.ReplaceAll(fileName, "|", "_")

	//Replace any whitespace
	fileName = strings.ReplaceAll(fileName, " ", "_")

	return fileName
}

// Contains returns true if the slice contains the string
func Contains(s []string, e string) bool {
	for _, a := range s {
		if strings.EqualFold(a, e) {
			return true
		}
	}
	return false
}

func SecondsToHHMMSS(seconds int) string {
	duration := time.Duration(seconds) * time.Second

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds = int(duration.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
