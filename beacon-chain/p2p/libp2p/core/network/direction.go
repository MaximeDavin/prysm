package network

import (
	"github.com/libp2p/go-libp2p/direction"
)

// DEPRECACTED Use `github.com/libp2p/go-libp2p/direction` instead
// Re-export direction for compatibility purpose
type Direction direction.Direction

const (
	// DEPRECACTED Use `github.com/libp2p/go-libp2p/direction` instead
	// DirUnknown is the default direction.
	DirUnknown Direction = iota
	// DEPRECACTED Use `github.com/libp2p/go-libp2p/direction` instead
	// DirInbound is for when the remote peer initiated a connection.
	DirInbound
	// DEPRECACTED Use `github.com/libp2p/go-libp2p/direction` instead
	// DirOutbound is for when the local peer initiated a connection.
	DirOutbound
)
