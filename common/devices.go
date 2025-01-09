package common

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	ThumbnailFolder = "system/thumbnails"
)

// NOTE: we keep zero terminator in the slice to avoid additional UTF16 to UTF16Ptr conversion
type PnPDeviceID []uint16

// VendorID returns Vendor ID or zero if it cannot parse PnP device ID string.
// NOTE: this is not what Microsoft recommends doing, descriptor is supposed to be opaque,
// but it is working on all Windows versions so far and alternative is bulky at best.
// https://learn.microsoft.com/en-us/windows-hardware/drivers/install/standard-usb-identifiers
func (p PnPDeviceID) VendorID() int {
	matches := regexp.MustCompile(`(?i)USB[#&]VID_([0-9A-F]+)&PID_[0-9A-F]+`).FindStringSubmatch(p.String())
	if len(matches) != 2 {
		return 0
	}
	var (
		id  int64
		err error
	)
	if id, err = strconv.ParseInt(matches[1], 16, 32); err != nil {
		return 0
	}
	return int(id)
}

// ProductID returns Product IDs or zero if it cannot parse PnP device ID string.
// NOTE: this is not what Microsoft recommends doing, descriptor is supposed to be opaque,
// but it is working on all Windows versions so far and alternative is bulky at best.
// https://learn.microsoft.com/en-us/windows-hardware/drivers/install/standard-usb-identifiers
func (p PnPDeviceID) ProductID() int {
	matches := regexp.MustCompile(`(?i)USB[#&]VID_[0-9A-F]+&PID_([0-9A-F]+)`).FindStringSubmatch(p.String())
	if len(matches) != 2 {
		return 0
	}
	var (
		id  int64
		err error
	)
	if id, err = strconv.ParseInt(matches[1], 16, 32); err != nil {
		return 0
	}
	return int(id)
}

// Serial returns SN for device or empty string if it cannot parse PnP device ID string.
// NOTE: this is not what Microsoft recommends doing, descriptor is supposed to be opaque,
// but it is working on all Windows versions so far and alternative is bulky at best.
func (p PnPDeviceID) Serial() string {
	matches := regexp.MustCompile(`(?i)USB[#&]VID_[0-9A-F]+&PID_[0-9A-F]+#(.+)#.+`).FindStringSubmatch(p.String())
	if len(matches) != 2 {
		return ""
	}
	return strings.ToUpper(matches[1])
}

func (p PnPDeviceID) String() string {
	return UTF16ToString(p)
}

type SupportedProtocols int

const (
	ProtocolUSB SupportedProtocols = iota
	ProtocolMTP
)

func (p SupportedProtocols) String() string {
	switch p {
	case ProtocolUSB:
		return "USB"
	case ProtocolMTP:
		return "MTP"
	default:
		return "Unknown"
	}
}

var supportedDevices = []struct {
	vid, pid int
	protocol SupportedProtocols
}{
	{0x1949, 0x0002, ProtocolUSB}, // Kindle
	{0x1949, 0x0004, ProtocolUSB}, // Kindle 3/4/Paperwhite
	{0x1949, 0x9981, ProtocolMTP}, // So far this is true for Kindle Scribe and Kindle Paperwhite MTP devices
}

func IsKindleDevice(protocol SupportedProtocols, vid, pid int) bool {
	if vid == 0 && pid == 0 {
		return false
	}
	for _, d := range supportedDevices {
		if d.protocol == protocol && d.vid == vid && d.pid == pid {
			return true
		}
	}
	return false
}
