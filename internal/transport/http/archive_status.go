package http

import (
	"fmt"
	"strings"

	"github.com/zibbp/ganymede/internal/utils"
)

func parseArchiveStatuses(value string) ([]utils.ArchiveStatus, error) {
	if value == "" {
		return nil, nil
	}
	var statuses []utils.ArchiveStatus
	for _, part := range strings.Split(value, ",") {
		status := utils.ArchiveStatus(part)
		if !status.Valid() {
			return nil, fmt.Errorf("invalid archive status %q", part)
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}
