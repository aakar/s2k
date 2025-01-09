package usb

import (
	"path/filepath"
	"unsafe"

	"go.uber.org/zap"
	"golang.org/x/sys/windows"

	"sync2kindle/common"
	"sync2kindle/files"
)

type Device struct {
	*files.Device
	log     *zap.Logger
	id      common.PnPDeviceID
	devinst windows.DEVINST
	eject   bool
}

func Connect(paths, serial string, eject bool, log *zap.Logger) (*Device, error) {

	var mount string
	id, mount, devinst, err := pickDevice(serial, log)
	if err != nil {
		return nil, err
	}

	d := &Device{log: log.Named(driverName), id: id, devinst: devinst, eject: eject}
	d.Device, err = files.Connect(paths, filepath.ToSlash(mount), nil, d.log)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// driver interface

type VetoType uint32

const (
	PNP_VetoTypeUnknown VetoType = iota
	PNP_VetoLegacyDevice
	PNP_VetoPendingClose
	PNP_VetoWindowsApp
	PNP_VetoWindowsService
	PNP_VetoOutstandingOpen
	PNP_VetoDevice
	PNP_VetoDriver
	PNP_VetoIllegalDeviceRequest
	PNP_VetoInsufficientPower
	PNP_VetoNonDisableable
	PNP_VetoLegacyDriver
	PNP_VetoInsufficientRights
	PNP_VetoAlreadyRemoved
)

func (v VetoType) String() string {
	switch v {
	case PNP_VetoLegacyDevice:
		return "The device does not support the specified PnP operation"
	case PNP_VetoPendingClose:
		return "The specified operation cannot be completed because of a pending close operation"
	case PNP_VetoWindowsApp:
		return "A Microsoft Win32 application vetoed the specified operation"
	case PNP_VetoWindowsService:
		return "A Win32 service vetoed the specified operation"
	case PNP_VetoOutstandingOpen:
		return "The requested operation was rejected because of outstanding open handles"
	case PNP_VetoDevice:
		return "The device supports the specified operation, but the device rejected the operation"
	case PNP_VetoDriver:
		return "The driver supports the specified operation, but the driver rejected the operation"
	case PNP_VetoIllegalDeviceRequest:
		return "The device does not support the specified operation"
	case PNP_VetoInsufficientPower:
		return "There is insufficient power to perform the requested operation"
	case PNP_VetoNonDisableable:
		return "The device cannot be disabled"
	case PNP_VetoLegacyDriver:
		return "The driver does not support the specified PnP operation"
	case PNP_VetoInsufficientRights:
		return "The caller has insufficient privileges to complete the operation"
	case PNP_VetoAlreadyRemoved:
		return "The device has already been removed"
	case PNP_VetoTypeUnknown:
		fallthrough
	default:
		return "The specified operation was rejected for an unknown reason"
	}
}

func (d *Device) Disconnect() {
	if d != nil && d.eject {
		vetoName := make([]uint16, windows.MAX_PATH)
		vetoType := VetoType(0)

		if rc, _, _ := cmRequestDeviceEject.Call(
			uintptr(d.devinst), uintptr(unsafe.Pointer(&vetoType)),
			uintptr(unsafe.Pointer(&vetoName[0])), uintptr(len(vetoName)), 0); windows.CONFIGRET(rc) != windows.CR_SUCCESS {
			if windows.CONFIGRET(rc) == windows.CR_REMOVE_VETOED {
				d.log.Warn("Eject failed", zap.Stringer("reason", vetoType))
				info := windows.UTF16ToString(vetoName)
				if len(info) > 0 {
					d.log.Debug("Eject failed", zap.String("Additional info", info))
				}
				return
			}
			d.log.Error("Eject failed", zap.Error(windows.CONFIGRET(rc)))
		}
	}
}

func (d *Device) Name() string {
	return driverName
}

func (d *Device) UniqueID() string {
	return d.id.Serial()
}
