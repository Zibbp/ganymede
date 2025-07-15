package exec

import (
	"fmt"
	"syscall"

	"github.com/rs/zerolog/log"
)

// killYtDlp attempts to gracefully terminate the yt-dlp process by sending a SIGINT to its process group.
func killYtDlp(pid int) error {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get process group ID for PID %d", pid)
		return fmt.Errorf("failed to get process group ID for PID %d: %w", pid, err)
	}
	log.Debug().Msgf("Process group ID for PID %d is %d", pid, pgid)
	return syscall.Kill(-pgid, syscall.SIGINT)
}
