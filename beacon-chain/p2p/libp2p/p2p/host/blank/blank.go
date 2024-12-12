package blank

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
)

// DEPRECATED
func NewBlankHost(n any, options ...any) *host.BHost {
	h, _ := libp2p.New()
	return h.(*host.BHost)
}
