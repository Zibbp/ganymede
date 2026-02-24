// Package storagetemplate provides shared storage template resolution logic
// for channel folder naming. This is extracted into a separate package to avoid
// circular dependencies between the archive and tasks packages.
package storagetemplate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/utils"
)

var (
	templateVariableRegex = regexp.MustCompile(`\{{([^}]+)\}}`)
)

// ChannelTemplateInput holds channel-specific variables for channel folder template resolution.
type ChannelTemplateInput struct {
	ChannelName        string // login name (lowercase username)
	ChannelID          string // external platform ID (e.g., Twitch User ID)
	ChannelDisplayName string // display name
}

// GetChannelFolderName resolves the channel-level folder name from the channel_folder_template config.
// The default template is "{{channel}}" which preserves backward compatibility.
// The resolved name is sanitized to prevent path traversal and invalid filesystem characters.
func GetChannelFolderName(input ChannelTemplateInput) (string, error) {
	channelTemplate := config.Get().StorageTemplates.ChannelFolderTemplate
	if channelTemplate == "" {
		log.Warn().Msg("Channel folder template is empty, falling back to channel login name")
		// Fallback to channel login name for backward compatibility
		channelTemplate = "{{channel}}"
	}

	variableMap := getChannelVariableMap(input)

	res := templateVariableRegex.FindAllStringSubmatch(channelTemplate, -1)
	for _, match := range res {
		variableName := match[1]
		variableValue, ok := variableMap[variableName]
		if !ok {
			return "", fmt.Errorf("variable %s not found in channel variable map", variableName)
		}
		valueString := fmt.Sprintf("%v", variableValue)
		if valueString == "" {
			return "", fmt.Errorf("variable %s resolved to empty string; ensure the channel has this field populated", variableName)
		}
		channelTemplate = strings.ReplaceAll(channelTemplate, match[0], valueString)
	}

	// Sanitize the resolved name to prevent path traversal (e.g., "../" or "/")
	// and invalid filesystem characters.
	channelTemplate = utils.SanitizeFileName(channelTemplate)

	if channelTemplate == "" {
		return "", fmt.Errorf("resolved channel folder name is empty after sanitization")
	}

	return channelTemplate, nil
}

func getChannelVariableMap(input ChannelTemplateInput) map[string]interface{} {
	safeDisplayName := utils.SanitizeFileName(input.ChannelDisplayName)

	return map[string]interface{}{
		"channel":              input.ChannelName,
		"channel_id":           input.ChannelID,
		"channel_display_name": safeDisplayName,
	}
}
