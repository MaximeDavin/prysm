package tcp

import (
	"context"

	"github.com/libp2p/go-libp2p/core/transport"
	"github.com/libp2p/go-libp2p/direction"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

type TcpListener struct {
	manet.Listener
	// Every accepted and upgraded connection in the listen loop's
	// goroutines are sent to this channel
	incoming chan transport.UpgradedConn
	// Store error happening during listen loop
	errs chan error
	// We keep a reference to the transport object to access muxers settings
	t *TcpTransport
}

var _ transport.Listener = &TcpListener{}

func NewTCPListener(l manet.Listener, t *TcpTransport) *TcpListener {
	return &TcpListener{
		Listener: l,
		incoming: make(chan transport.UpgradedConn),
		errs:     make(chan error),
		t:        t,
	}
}

// Listen returns a listener that listens on the given multiaddr for inbound
// connections.
//
// For each connection accepted by the net.listener, a goroutine is run to
// upgrade the connection (adding security and stream multiplexing).
// Then, the upgraded connection is sent to the 'incoming' channel
// and received by the 'Accept' method of the returned listener.
// The use of goroutines allows the listener to keep accepting new connections
// while a connection is upgraded.
func (t *TcpTransport) Listen(addr ma.Multiaddr) (transport.Listener, error) {
	l, err := manet.Listen(addr)
	if err != nil {
		return nil, err
	}

	tcpl := NewTCPListener(l, t)
	go tcpl.Serve()

	return tcpl, nil
}

// Accepts incoming connections from the listen loop
func (l *TcpListener) Accept() (transport.UpgradedConn, error) {
	select {
	case c := <-l.incoming:
		return c, nil
	case err := <-l.errs:
		return nil, err
	}
}

// Accepts connections on the listener and upgrade each incoming connections
func (l *TcpListener) Serve() {
	defer l.Close()
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			// Temporary errors could be retried here
			// see https://go.dev/src/net/http/server.go?s=51504:51550#L3335
			// However it has been deprecated see https://groups.google.com/g/golang-nuts/c/-JcZzOkyqYI/m/xwaZzjCgAwAJ
			l.errs <- err
			return
		}
		go l.upgrade(conn)
	}
}

// Upgrade the raw connection to a connection that support security and multiplexing
func (l *TcpListener) upgrade(conn manet.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), l.t.AcceptTimeout)
	defer cancel()

	sconn, err := Upgrade(ctx, l.t, conn, "", direction.DirInbound)
	if err != nil {
		l.errs <- err
		return
	}

	select {
	case l.incoming <- sconn:
	case <-ctx.Done():
		conn.Close()
	}
}

func (l *TcpListener) Close() error {
	return l.Listener.Close()
}
