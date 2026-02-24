package ego

import "errors"

// ErrNotFound is returned when a query expects at least one result but finds none.
var ErrNotFound = errors.New("ego: entity not found")
