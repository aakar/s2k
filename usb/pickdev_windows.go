package usb

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	"go.uber.org/zap"
	"golang.org/x/sys/windows"

	"sync2kindle/common"
)

var (
	modsetupapi = windows.NewLazySystemDLL("setupapi.dll")

	setupDiGetClassDevs             = modsetupapi.NewProc("SetupDiGetClassDevsW")
	setupDiEnumDeviceInterfaces     = modsetupapi.NewProc("SetupDiEnumDeviceInterfaces")
	setupDiGetDeviceInterfaceDetail = modsetupapi.NewProc("SetupDiGetDeviceInterfaceDetailW")
	setupDiDestroyDeviceInfoList    = modsetupapi.NewProc("SetupDiDestroyDeviceInfoList")

	modecmgr = windows.NewLazySystemDLL("cfgmgr32.dll")

	cmGetChild           = modecmgr.NewProc("CM_Get_Child")
	cmGetDeviceIDSize    = modecmgr.NewProc("CM_Get_Device_ID_Size")
	cmGetDeviceID        = modecmgr.NewProc("CM_Get_Device_IDW")
	cmRequestDeviceEject = modecmgr.NewProc("CM_Request_Device_EjectW")
)

type DevInfoData struct {
	size      uint32
	ClassGUID windows.GUID
	DevInst   windows.DEVINST
	_         uintptr
}

type DeviceInterfaceData struct {
	cbSize             uint32
	InterfaceClassGuid windows.GUID
	Flags              uint32
	_                  uintptr
}

type DeviceInterfaceDetailData struct {
	cbSize     uint32
	DevicePath [1]uint16 // ANYSIZE_ARRAY
}

var (
	GUID_DEVINTERFACE_USB_DEVICE = ole.NewGUID("a5dcbf10-6530-11d2-901f-00c04fb951ed")
	GUID_DEVINTERFACE_DISK       = ole.NewGUID("53f56307-b6bf-11d0-94f2-00a0c91efb8b")
	GUID_DEVINTERFACE_VOLUME     = ole.NewGUID("53f5630d-b6bf-11d0-94f2-00a0c91efb8b")
)

func pickDevice(serial string, log *zap.Logger) (common.PnPDeviceID, string, windows.DEVINST, error) {

	guid := GUID_DEVINTERFACE_USB_DEVICE

	hDevInfo, _, err := setupDiGetClassDevs.Call(uintptr(unsafe.Pointer(guid)), uintptr(0), uintptr(0),
		uintptr(windows.DIGCF_PRESENT|windows.DIGCF_DEVICEINTERFACE))
	if windows.Handle(hDevInfo) == windows.InvalidHandle {
		return nil, "", windows.DEVINST(0), fmt.Errorf("unable to enumerate disk devices: %w", err)
	}
	defer setupDiDestroyDeviceInfoList.Call(hDevInfo)

	var (
		result     common.PnPDeviceID
		devinst    windows.DEVINST
		volumePath string
		dwSize     uint32
	)
	for dwIndex := 0; ; dwIndex++ {
		spdid := &DeviceInterfaceData{}
		spdid.cbSize = uint32(unsafe.Sizeof(*spdid))

		res, _, err := setupDiEnumDeviceInterfaces.Call(hDevInfo,
			uintptr(0), uintptr(unsafe.Pointer(guid)), uintptr(dwIndex), uintptr(unsafe.Pointer(spdid)))
		if res == 0 {
			var errno syscall.Errno
			if errors.As(err, &errno) && errno == windows.ERROR_NO_MORE_ITEMS {
				break
			}
			return nil, "", windows.DEVINST(0), fmt.Errorf("unable to enumerate disk device interfaces: %w", err)
		}

		setupDiGetDeviceInterfaceDetail.Call(hDevInfo, uintptr(unsafe.Pointer(spdid)), uintptr(0), uintptr(0), uintptr(unsafe.Pointer(&dwSize)), uintptr(0))
		if dwSize <= 0 {
			continue
		}

		buf := make([]byte, dwSize)
		spdidd := (*DeviceInterfaceDetailData)(unsafe.Pointer(&buf[0]))
		spdidd.cbSize = uint32(unsafe.Sizeof(spdidd))

		spdd := &DevInfoData{}
		spdd.size = uint32(unsafe.Sizeof(*spdd))

		res, _, err = setupDiGetDeviceInterfaceDetail.Call(hDevInfo,
			uintptr(unsafe.Pointer(spdid)), uintptr(unsafe.Pointer(spdidd)), uintptr(dwSize), uintptr(unsafe.Pointer(&dwSize)), uintptr(unsafe.Pointer(spdd)))
		if res == 0 {
			return nil, "", windows.DEVINST(0), fmt.Errorf("unable to get disk device interface details: %w", err)
		}

		id := common.PnPDeviceID(common.UTF16PtrToUTF16(&spdidd.DevicePath[0]))
		vid, pid := id.VendorID(), id.ProductID()
		supported := common.IsKindleDevice(common.ProtocolUSB, vid, pid)
		log.Debug("Driver Info",
			zap.Stringer("PnP ID", id),
			zap.Bool("supported", supported),
		)

		if !supported {
			continue
		}

		if len(serial) > 0 {
			if !strings.EqualFold(serial, id.Serial()) {
				continue
			}
			// we are targeting a specific device
		} else {
			if len(result) != 0 {
				continue
			}
			// Pick first supported device
		}
		result = id
		devinst = spdd.DevInst

		var disk windows.DEVINST
		if rc, _, _ := cmGetChild.Call(uintptr(unsafe.Pointer(&disk)), uintptr(spdd.DevInst), 0); windows.CONFIGRET(rc) != windows.CR_SUCCESS {
			return nil, "", windows.DEVINST(0), fmt.Errorf("unable to find kindle disk device: %w", windows.CONFIGRET(rc))
		}
		idSize := uint32(0)
		if rc, _, _ := cmGetDeviceIDSize.Call(uintptr(unsafe.Pointer(&idSize)), uintptr(disk), 0); windows.CONFIGRET(rc) != windows.CR_SUCCESS {
			return nil, "", windows.DEVINST(0), fmt.Errorf("unable to get kindle disk device ID size: %w", windows.CONFIGRET(rc))
		}
		diskID := make([]uint16, idSize+1)
		if rc, _, _ := cmGetDeviceID.Call(uintptr(disk), uintptr(unsafe.Pointer(&diskID[0])), uintptr(len(diskID)), 0); windows.CONFIGRET(rc) != windows.CR_SUCCESS {
			return nil, "", windows.DEVINST(0), fmt.Errorf("unable to get kindle disk device ID: %w", windows.CONFIGRET(rc))
		}

		volume, err := findCorrespondingVolume(diskID)
		if err != nil {
			return nil, "", windows.DEVINST(0), err
		}

		if volumePath, err = getVolumePath(volume); err != nil {
			return nil, "", windows.DEVINST(0), fmt.Errorf("unable to get volume root path for volume '%s': %w", windows.UTF16ToString(volume), err)
		}

		log.Debug("Device Info",
			zap.Stringer("Device ID", result),
			zap.String("VendorID", fmt.Sprintf("0x%04X", vid)),
			zap.String("ProductID", fmt.Sprintf("0x%04X", pid)),
			zap.String("Serial", id.Serial()),
			zap.String("Disk ID", windows.UTF16ToString(diskID)),
			zap.String("Volume", windows.UTF16ToString(volume)),
			zap.String("Mount path", volumePath),
		)
	}
	if len(result) == 0 || len(volumePath) == 0 {
		return nil, "", windows.DEVINST(0), common.ErrNoDevice
	}
	return result, volumePath, devinst, nil
}

func findCorrespondingVolume(parentID []uint16) ([]uint16, error) {

	guid := GUID_DEVINTERFACE_VOLUME

	hDevInfo, _, err := setupDiGetClassDevs.Call(uintptr(unsafe.Pointer(guid)), uintptr(0), uintptr(0),
		uintptr(windows.DIGCF_PRESENT|windows.DIGCF_DEVICEINTERFACE))
	if windows.Handle(hDevInfo) == windows.InvalidHandle {
		return nil, fmt.Errorf("unable to enumerate volumes: %w", err)
	}
	defer setupDiDestroyDeviceInfoList.Call(hDevInfo)

	var dwSize uint32
	for dwIndex := 0; ; dwIndex++ {
		spdid := &DeviceInterfaceData{}
		spdid.cbSize = uint32(unsafe.Sizeof(*spdid))

		res, _, err := setupDiEnumDeviceInterfaces.Call(hDevInfo,
			uintptr(0), uintptr(unsafe.Pointer(guid)), uintptr(dwIndex), uintptr(unsafe.Pointer(spdid)))
		if res == 0 {
			var errno syscall.Errno
			if errors.As(err, &errno) && errno == windows.ERROR_NO_MORE_ITEMS {
				break
			}
			return nil, fmt.Errorf("unable to enumerate volume device interfaces: %w", err)
		}

		setupDiGetDeviceInterfaceDetail.Call(hDevInfo, uintptr(unsafe.Pointer(spdid)), uintptr(0), uintptr(0), uintptr(unsafe.Pointer(&dwSize)), uintptr(0))
		if dwSize <= 0 {
			continue
		}

		buf := make([]byte, dwSize)
		spdidd := (*DeviceInterfaceDetailData)(unsafe.Pointer(&buf[0]))
		spdidd.cbSize = uint32(unsafe.Sizeof(spdidd))

		spdd := &DevInfoData{}
		spdd.size = uint32(unsafe.Sizeof(*spdd))

		res, _, err = setupDiGetDeviceInterfaceDetail.Call(hDevInfo,
			uintptr(unsafe.Pointer(spdid)), uintptr(unsafe.Pointer(spdidd)), uintptr(dwSize), uintptr(unsafe.Pointer(&dwSize)), uintptr(unsafe.Pointer(spdd)))
		if res == 0 {
			return nil, fmt.Errorf("unable to get volume device interface details: %w", err)
		}

		mountPoint := common.UTF16PtrToUTF16(&spdidd.DevicePath[0])
		if strings.Contains(windows.UTF16ToString(mountPoint), strings.ReplaceAll(strings.ToLower(windows.UTF16ToString(parentID)), `\`, `#`)) {
			mountPoint = append(mountPoint[:len(mountPoint)-1], uint16('\\'), 0)
			volumeName := make([]uint16, windows.MAX_PATH+1)
			if err := windows.GetVolumeNameForVolumeMountPoint(&mountPoint[0], &volumeName[0], uint32(len(volumeName))); err != nil {
				return nil, fmt.Errorf("unable to get volume name for mount point '%s', %w", windows.UTF16ToString(mountPoint), err)
			}
			return common.UTF16PtrToUTF16(&volumeName[0]), nil
		}
	}
	return nil, common.ErrNoDevice
}

func getVolumePath(volume []uint16) (string, error) {
	var err error

	size := uint32(windows.MAX_PATH + 1)
	names := make([]uint16, size)

	for {
		if err = windows.GetVolumePathNamesForVolumeName(&volume[0], &names[0], size, &size); err == nil {
			break
		}
		if err != syscall.ERROR_MORE_DATA {
			return "", fmt.Errorf("unable to get volume path names for volume '%s': %w", windows.UTF16ToString(volume), err)
		}
		names = make([]uint16, size)
	}

	var paths = []string{}
	start := 0
	for end := 0; end < int(size); end++ {
		if names[end] == 0 {
			if start < end {
				paths = append(paths, windows.UTF16ToString(names[start:end]))
			}
			start = end + 1
		}
	}
	if len(paths) == 0 || len(paths) > 1 {
		return "", fmt.Errorf("ambiguous volume path names found for volume '%s' : '%+v'", windows.UTF16ToString(volume), paths)
	}
	return paths[0], nil
}
