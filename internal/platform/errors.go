package platform

import "errors"

type ErrorNoStreamsFound struct{}

func (e ErrorNoStreamsFound) Error() string {
	return "no streams found"
}

var ErrNotImplemented = errors.New("not implemented")
