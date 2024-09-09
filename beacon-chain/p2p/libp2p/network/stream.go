package network

import (
	"sync/atomic"

	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/transport"
)

type Stream struct {
	stream   transport.Stream
	conn     *transport.Conn
	protocol atomic.Pointer[protocol.ID]
}

// Protocol returns the protocol negotiated on this stream (if set).
func (s *Stream) Protocol() protocol.ID {
	p := s.protocol.Load()
	if p == nil {
		return ""
	}
	return *p
}

// SetProtocol sets the protocol for this stream.
func (s *Stream) SetProtocol(p protocol.ID) error {
	s.protocol.Store(&p)
	return nil
}

type StreamHandler func(Stream)
