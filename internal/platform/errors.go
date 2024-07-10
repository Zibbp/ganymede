package platform

import "fmt"

type ErrorNoStreamsFound struct{}

func (e ErrorNoStreamsFound) Error() string {
	return fmt.Sprintf("no streams found")
}
