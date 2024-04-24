package utils

type ErrorResponse struct {
	Message string `json:"message"`
}

type LiveVideoDownloadNoStreamError struct {
	message string
}

func (e *LiveVideoDownloadNoStreamError) Error() string {
	return e.message
}

func NewLiveVideoDownloadNoStreamError(message string) error {
	return &LiveVideoDownloadNoStreamError{message: message}
}
