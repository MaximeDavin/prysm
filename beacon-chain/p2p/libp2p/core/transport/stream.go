package transport

import (
	"io"
	"time"
)

// A bidirectional io pipe within a connection
type Stream interface {
	io.Reader
	io.Writer
	io.Closer

	CloseWrite() error
	Reset() error

	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}
