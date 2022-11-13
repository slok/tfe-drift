package internalerrors

import "fmt"

var (
	ErrNotExist      = fmt.Errorf("resource does not exist")
	ErrDriftDetected = fmt.Errorf("drifts detected")
)
