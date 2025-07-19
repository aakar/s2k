package mtp

import (
	"errors"
	"go.uber.org/zap"
	"sync2kindle/objects"
)

// Device stub for macOS.
type Device struct{}

// Connect returns an error indicating MTP is not supported on macOS.
func Connect(paths, serial string, verbose bool, _ *zap.Logger) (*Device, error) {
	return nil, errors.New("MTP support not implemented on darwin")
}

// Stub implementations to satisfy driver interface.
func (d *Device) Disconnect()                      {}
func (d *Device) Name() string                     { return driverName }
func (d *Device) UniqueID() string                 { return "" }
func (d *Device) MkDir(*objects.ObjectInfo) error  { return errors.New("not supported") }
func (d *Device) Remove(*objects.ObjectInfo) error { return errors.New("not supported") }
func (d *Device) Copy(*objects.ObjectInfo) error   { return errors.New("not supported") }
func (d *Device) GetObjectInfos() (objects.ObjectInfoSet, error) {
	return nil, errors.New("not supported")
}
