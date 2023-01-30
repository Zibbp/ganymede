package utils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var (
	allowedLogTypes = []string{"video", "video-convert", "chat", "chat-render", "chat-convert"}
)

type CustomValidator struct {
	Validator *validator.Validate
}

// Init validator
func (cv *CustomValidator) Init() {
}

// Validate Data
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.Validator.Struct(i)
}

func ValidateLogType(logType string) (string, error) {
	if !IsValidLogType(logType) {
		return "", fmt.Errorf("invalid log type: %s", logType)
	}
	return logType, nil
}

func IsValidLogType(logType string) bool {
	for _, t := range allowedLogTypes {
		if t == logType {
			return true
		}
	}
	return false
}

func IsValidUUID(input string) (uuid.UUID, error) {
	id, err := uuid.Parse(input)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func ValidateFileNameInput(input string) (string, error) {
	disallowedChars := []string{"/", "\\", ".", ".."}
	for _, disallowedChar := range disallowedChars {
		if strings.Contains(input, disallowedChar) {
			return "", fmt.Errorf("input contains disallowed character: %s", disallowedChar)
		}
	}

	validFileNameChars := `^[^/\\:*?"<>|]+$`
	re := regexp.MustCompile(validFileNameChars)
	if !re.MatchString(input) {
		return "", fmt.Errorf("input contains characters that are not valid for a file name")
	}

	return input, nil
}

func ValidateFileName(fileName string) (string, error) {
	if strings.Count(fileName, ".") > 1 {
		return "", fmt.Errorf("file name contains more than one '.' character")
	}

	if strings.ContainsAny(fileName, "/\\") {
		return "", fmt.Errorf("file name contains a directory separator character")
	}

	validFileNamePattern := `^[^/\\:*?"<>|]+$`
	re := regexp.MustCompile(validFileNamePattern)
	if !re.MatchString(fileName) {
		return "", fmt.Errorf("file name does not match allowed pattern")
	}

	return fileName, nil
}
