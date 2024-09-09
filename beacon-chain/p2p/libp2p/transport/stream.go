package transport

import (
	"io"
	"time"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/protocol"
)

// A bidirectional io pipe within a connection
type Stream interface {
	io.Reader
	io.Writer
	io.Closer

	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

type stream struct {
	Stream
	Protocol
}
type Protocol struct {
	id *protocol.ID
}

func (p Protocol) SetProtocol(id protocol.ID) {
	p.id = &id
}

func (p Protocol) Protocol() protocol.ID {
	return *p.id
}

type StreamHandler func(Stream)
