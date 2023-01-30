package utils

import (
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
