package common

import (
	"errors"
)

var (
	ErrNoDevice  = errors.New("no available device found")
	ErrNoStorage = errors.New("no storage found on the device")
	ErrNoAccess  = errors.New("no write access to the device storage")
	ErrNoObjects = errors.New("no objects found on the device")
	ErrNoFiles   = errors.New("no files found")
)
