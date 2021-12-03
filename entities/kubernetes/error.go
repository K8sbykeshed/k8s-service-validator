package kubernetes

import "errors"

var ErrLabelNotFound = errors.New("label not found")

func IsLabelNotFound(err error) bool {
	return err == ErrLabelNotFound
}
