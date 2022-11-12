package internalerrors

import "fmt"

var (
	ErrNotExist = fmt.Errorf("resource does not exist")
)
