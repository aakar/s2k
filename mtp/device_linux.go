package mtp

import (
	ole "github.com/go-ole/go-ole"
	"go.uber.org/zap"

	"sync2kindle/common"
	"sync2kindle/objects"
)

type Device struct {
}

func Connect(_, _ string, _ *zap.Logger) (*Device, error) {
	return nil, common.ErrNoDevice
}

func (d *Device) Disconnect() {
}

func (d *Device) UniqueID() string {
	return ""
}

// driver interface

func (d *Device) Name() string {
	return driverName
}

func (d *Device) GetObjectInfos() (objects.ObjectInfoSet, error) {
	return nil, common.ErrNoObjects
}

func (d *Device) MkDir(obj *objects.ObjectInfo) error {
	return ole.NewError(ole.E_NOTIMPL)
}

func (d *Device) Remove(obj *objects.ObjectInfo) error {
	return ole.NewError(ole.E_NOTIMPL)
}

func (d *Device) Copy(obj *objects.ObjectInfo) error {
	return ole.NewError(ole.E_NOTIMPL)
}
