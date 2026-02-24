package archive

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/storagetemplate"
	"github.com/zibbp/ganymede/internal/utils"
)

var (
	storageTemplateVariableRegex = regexp.MustCompile(`\{{([^}]+)\}}`)
)

type StorageTemplateInput struct {
	UUID               uuid.UUID
	ID                 string
	Channel            string
	ChannelID          string // external platform ID (e.g., Twitch User ID)
	ChannelDisplayName string // channel display name
	Title              string
	Type               string
	Date               string // parsed date
	YYYY               string // year
	MM                 string // month
	DD                 string // day
	HH                 string // hour
}

// ChannelTemplateInput is an alias for storagetemplate.ChannelTemplateInput
// so callers in the archive package can use it without importing storagetemplate directly.
type ChannelTemplateInput = storagetemplate.ChannelTemplateInput

func GetFolderName(uuid uuid.UUID, input StorageTemplateInput) (string, error) {

	variableMap, err := getVariableMap(uuid, input)
	if err != nil {
		return "", fmt.Errorf("error getting variable map: %w", err)
	}

	folderTemplate := config.Get().StorageTemplates.FolderTemplate
	if folderTemplate == "" {
		log.Error().Msg("Folder template is empty")
		// Fallback template
		folderTemplate = "{{id}}-{{uuid}}"
	}

	res := storageTemplateVariableRegex.FindAllStringSubmatch(folderTemplate, -1)
	for _, match := range res {
		// Get variable name
		variableName := match[1]
		// Get variable value
		variableValue, ok := variableMap[variableName]
		if !ok {
			return "", fmt.Errorf("variable %s not found in variable map", variableName)
		}
		// Replace variable in template
		folderString := fmt.Sprintf("%v", variableValue)
		folderTemplate = strings.ReplaceAll(folderTemplate, match[0], folderString)

	}

	return folderTemplate, nil
}

func GetFileName(uuid uuid.UUID, input StorageTemplateInput) (string, error) {

	variableMap, err := getVariableMap(uuid, input)
	if err != nil {
		return "", fmt.Errorf("error getting variable map: %w", err)
	}

	fileTemplate := config.Get().StorageTemplates.FileTemplate
	if fileTemplate == "" {
		log.Error().Msg("File template is empty")
		// Fallback template
		fileTemplate = "{{id}}"
	}

	res := storageTemplateVariableRegex.FindAllStringSubmatch(fileTemplate, -1)
	for _, match := range res {
		// Get variable name
		variableName := match[1]
		// Get variable value
		variableValue, ok := variableMap[variableName]
		if !ok {
			return "", fmt.Errorf("variable %s not found in variable map", variableName)
		}
		// Replace variable in template
		fileString := fmt.Sprintf("%v", variableValue)
		fileTemplate = strings.ReplaceAll(fileTemplate, match[0], fileString)

	}

	return fileTemplate, nil
}

// GetChannelFolderName resolves the channel-level folder name from the channel_folder_template config.
// The default template is "{{channel}}" which preserves backward compatibility.
// This delegates to the storagetemplate package so that other packages (e.g. tasks)
// can also resolve channel folder names without importing archive.
func GetChannelFolderName(input ChannelTemplateInput) (string, error) {
	return storagetemplate.GetChannelFolderName(input)
}

func getVariableMap(uuid uuid.UUID, input StorageTemplateInput) (map[string]interface{}, error) {
	safeTitle := utils.SanitizeFileName(input.Title)

	variables := map[string]interface{}{
		"uuid":                 uuid.String(),
		"id":                   input.ID,
		"channel":              input.Channel,
		"channel_id":           input.ChannelID,
		"channel_display_name": input.ChannelDisplayName,
		"title":                safeTitle,
		"date":                 input.Date,
		"type":                 input.Type,
		"YYYY":                 input.YYYY,
		"MM":                   input.MM,
		"DD":                   input.DD,
		"HH":                   input.HH,
	}
	return variables, nil
}
