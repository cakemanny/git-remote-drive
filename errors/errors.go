package errors

import (
	builtinerrors "errors"
	"fmt"
)

var ErrNotImplemented = builtinerrors.New("not implemented")

// ErrNotFound is returned when a file or directory does not exist
type ErrNotFound struct {
	Path string
}

type ErrInvalidObject struct {
	Sha string
}

func (err ErrNotFound) Error() string {
	return fmt.Sprintf("%s: no such file or directory", err.Path)
}

func (err ErrInvalidObject) Error() string {
	return fmt.Sprintf("%s: sha1 of contents does not match", err.Sha)
}

func NotImplemented() error {
	return ErrNotImplemented
}
