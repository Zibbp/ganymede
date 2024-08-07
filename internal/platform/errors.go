package platform

type ErrorNoStreamsFound struct{}

func (e ErrorNoStreamsFound) Error() string {
	return "no streams found"
}
