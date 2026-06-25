package storage

import "errors"

var (
	ErrProfileNameRequired  = errors.New("profile name is required")
	ErrDuplicateProfileName = errors.New("duplicate profile name")
	ErrDuplicateTagName     = errors.New("duplicate tag name")
	ErrModNotFound          = errors.New("mod not found")
	ErrProfileNotFound      = errors.New("profile not found")
)
