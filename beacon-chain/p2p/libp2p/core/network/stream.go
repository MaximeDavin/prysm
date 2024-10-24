package network

import (
	"time"

	"libp2p/core/protocol"
	"libp2p/core/transport"
)

type Stream struct {
	stream   transport.Stream
	conn     *Conn
	protocol protocol.ID
	stat     Stats
}

func NewStream(s transport.Stream, conn *Conn, proto protocol.ID, direction transport.Direction) *Stream {
	stream := &Stream{
		stream: s,
		conn:   conn,
		stat:   Stats{Direction: direction},
	}
	stream.SetProtocol(proto)
	return stream
}

// Conn returns the Conn associated with this stream, as an network.Conn
func (s *Stream) Conn() *Conn {
	return s.conn
}

// Read reads bytes from a stream.
func (s *Stream) Read(p []byte) (int, error) {
	n, err := s.stream.Read(p)
	return n, err
}

// Write writes bytes to a stream, flushing for each call.
func (s *Stream) Write(p []byte) (int, error) {
	n, err := s.stream.Write(p)
	return n, err
}

// Close closes the stream, closing both ends and freeing all associated
// resources.
func (s *Stream) Close() error {
	return s.stream.Close()
}

// Reset resets the stream, signaling an error on both ends and freeing all
// associated resources.
func (s *Stream) Reset() error {
	err := s.stream.Reset()
	return err
}

// CloseWrite closes the stream for writing, flushing all data and sending an EOF.
// This function does not free resources, call Close or Reset when done with the
// stream.
func (s *Stream) CloseWrite() error {
	return s.stream.CloseWrite()
}

// Protocol returns the protocol negotiated on this stream (if set).
func (s *Stream) Protocol() protocol.ID {
	return s.protocol
}

// SetProtocol sets the protocol for this stream.
//
// This doesn't actually *do* anything other than record the fact that we're
// speaking the given protocol over this stream. It's still up to the user to
// negotiate the protocol. This is usually done by the Host.
func (s *Stream) SetProtocol(p protocol.ID) {
	s.protocol = p
}

// SetDeadline sets the read and write deadlines for this stream.
func (s *Stream) SetDeadline(t time.Time) error {
	return s.stream.SetDeadline(t)
}

// SetReadDeadline sets the read deadline for this stream.
func (s *Stream) SetReadDeadline(t time.Time) error {
	return s.stream.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline for this stream.
func (s *Stream) SetWriteDeadline(t time.Time) error {
	return s.stream.SetWriteDeadline(t)
}

// Stat returns metadata information for this stream.
func (s *Stream) Stat() Stats {
	return s.stat
}

type StreamHandler func(transport.Stream)
