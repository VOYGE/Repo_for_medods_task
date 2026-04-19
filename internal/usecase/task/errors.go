package task

import "errors"

var ErrInvalidInput = errors.New("invalid task input")

var ErrNotATemplate = errors.New("task is not a recurrence template")
