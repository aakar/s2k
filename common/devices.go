package common

const (
	ThumbnailFolder = "system/thumbnails"
)

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
