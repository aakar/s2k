package history

import (
	"crypto/sha256"
	"fmt"
	"io"

	"sync2kindle/common"
)

func GetName(protocol common.SupportedProtocols, ids ...string) string {
	h := sha256.New()
	for _, id := range ids {
		_, _ = io.WriteString(h, id)
	}
	_, _ = io.WriteString(h, protocol.String())
	return fmt.Sprintf("%x.db", h.Sum(nil))
}
