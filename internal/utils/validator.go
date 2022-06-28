package utils

import "github.com/go-playground/validator/v10"

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
