package transport

import (
	"io"
	"time"
)

// A bidirectional io pipe within a connection
type MuxedStream interface {
	io.Reader
	io.Writer
	io.Closer

	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}
