//go:build !windows

package common

import (
	"fmt"
)

type PnPDeviceID struct {
	VID, PID, BCD int
	Serial        string
}

func (id PnPDeviceID) String() string {
	return fmt.Sprintf("USB&VID_%04X&PID_%04X#%s#%02d.%d%d", id.VID, id.PID, id.Serial, id.BCD>>8, id.BCD&0xF0, id.BCD&0x0F)
}
