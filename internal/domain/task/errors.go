package task

import "errors"

var ErrNotFound = errors.New("task not found")

var ErrInvalidRecurrence = errors.New("invalid recurrence rule")
