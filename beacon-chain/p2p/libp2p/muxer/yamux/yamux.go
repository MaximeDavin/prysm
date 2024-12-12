package yamux

import (
	"context"

	"github.com/libp2p/go-libp2p/core/transport"
	"github.com/libp2p/go-libp2p/direction"

	"github.com/hashicorp/yamux"
)

var DefaultTransport *Transport

const ID = "/yamux/1.0.0"

// Transport implements mux.Multiplexer that constructs
// yamux-backed muxed connections.
type Transport yamux.Config

func Multiplex(conn transport.SecureConn, dir direction.Direction) (transport.MuxedConn, error) {
	var session *yamux.Session
	var err error
	if dir == direction.DirOutbound {
		session, err = yamux.Client(conn, nil)
	} else {
		session, err = yamux.Server(conn, nil)
	}
	if err != nil {
		return nil, err
	}
	return &yamuxConn{Session: session}, nil
}

type yamuxConn struct {
	*yamux.Session
}

var _ transport.MuxedConn = &yamuxConn{}

func (y *yamuxConn) OpenStream(ctx context.Context) (transport.Stream, error) {
	stream, err := y.Session.OpenStream()
	s := NewStream(stream)
	return transport.Stream(s), err
}

func (y *yamuxConn) AcceptStream() (transport.Stream, error) {
	stream, err := y.Session.AcceptStream()
	s := NewStream(stream)
	return transport.Stream(s), err
}

func (y *yamuxConn) Close() error {
	return y.Session.Close()
}

func (y *yamuxConn) IsClosed() bool {
	return y.Session.IsClosed()
}
