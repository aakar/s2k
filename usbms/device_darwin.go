package usbms

import (
	"errors"

	"go.uber.org/zap"
	"sync2kindle/objects"
)

type Device struct{}

// Connect to the supported device. Currently not implemented for macOS.
func Connect(paths, serial string, eject bool, _ *zap.Logger) (*Device, error) {
	return nil, errors.New("USBMS support not implemented on darwin")
}

// Stub implementations to satisfy the driver interface.
func (d *Device) Disconnect()                      {}
func (d *Device) Name() string                     { return driverName }
func (d *Device) UniqueID() string                 { return "" }
func (d *Device) MkDir(*objects.ObjectInfo) error  { return errors.New("not supported") }
func (d *Device) Remove(*objects.ObjectInfo) error { return errors.New("not supported") }
func (d *Device) Copy(*objects.ObjectInfo) error   { return errors.New("not supported") }
func (d *Device) GetObjectInfos() (objects.ObjectInfoSet, error) {
	return nil, errors.New("not supported")
}
