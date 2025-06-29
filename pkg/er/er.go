package er

import "errors"

var (
	ErrOrderNotFound = errors.New("order not found")
	ErrOrderExists   = errors.New("order already exists")
	ErrInvalidData   = errors.New("invalid data")
	ErrDatabaseError = errors.New("database error")
)
