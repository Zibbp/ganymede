package archive

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/internal/utils"
)

var (
	storageTemplateVariableRegex = regexp.MustCompile(`\{{([^}]+)\}}`)
)

type StorageTemplateInput struct {
	UUID    uuid.UUID
	ID      string
	Channel string
	Title   string
	Type    string
	Date    string // parsed date
}

func GetFolderName(uuid uuid.UUID, input StorageTemplateInput) (string, error) {

	variableMap, err := getVariableMap(uuid, input)
	if err != nil {
		return "", fmt.Errorf("error getting variable map: %w", err)
	}

	folderTemplate := viper.GetString("storage_templates.folder_template")
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
		folderTemplate = strings.Replace(folderTemplate, match[0], folderString, -1)

	}

	return folderTemplate, nil
}

func GetFileName(uuid uuid.UUID, input StorageTemplateInput) (string, error) {

	variableMap, err := getVariableMap(uuid, input)
	if err != nil {
		return "", fmt.Errorf("error getting variable map: %w", err)
	}

	fileTemplate := viper.GetString("storage_templates.file_template")
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
		fileTemplate = strings.Replace(fileTemplate, match[0], fileString, -1)

	}

	return fileTemplate, nil
}

func getVariableMap(uuid uuid.UUID, input StorageTemplateInput) (map[string]interface{}, error) {
	safeTitle := utils.SanitizeFileName(input.Title)

	variables := map[string]interface{}{
		"uuid":    uuid.String(),
		"id":      input.ID,
		"channel": input.Channel,
		"title":   safeTitle,
		"date":    input.Date,
		"type":    input.Type,
	}
	return variables, nil
}
