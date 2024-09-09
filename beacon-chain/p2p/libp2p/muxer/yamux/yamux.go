package yamux

import (
	"context"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/transport"

	"github.com/hashicorp/yamux"
)

// var DefaultTransport *Transport

const ID = "/yamux/1.0.0"

// Transport implements mux.Multiplexer that constructs
// yamux-backed muxed connections.
type Transport yamux.Config

func Multiplex(conn transport.SecureConn, direction transport.Direction) (transport.MuxedConn, error) {
	var session *yamux.Session
	var err error
	if direction == transport.DirOutbound {
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
	return transport.Stream(stream), err
}

func (y *yamuxConn) AcceptStream() (transport.Stream, error) {
	stream, err := y.Session.AcceptStream()
	return transport.Stream(stream), err
}

func (y *yamuxConn) Close() error {
	return y.Session.Close()
}

func (y *yamuxConn) IsClosed() bool {
	return y.Session.IsClosed()
}
