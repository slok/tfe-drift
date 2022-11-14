package internalerrors

import "fmt"

var (
	ErrNotExist                 = fmt.Errorf("resource does not exist")
	ErrDriftDetected            = fmt.Errorf("drift detected")
	ErrDriftDetectionPlanFailed = fmt.Errorf("drift detection plan failed")
)
