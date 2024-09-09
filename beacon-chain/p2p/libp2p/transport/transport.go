package transport

import (
	"context"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/peer"
)

// The Transport interface allows you to open connections to other peers
// by dialing them, and also lets you listen for incoming connections.
//
// Connections returned by Dial and passed into Listeners are of type
// CapableConn, which means that they have been upgraded to support
// stream multiplexing and connection security (encryption and authentication).
type Transport interface {
	Dial(ctx context.Context, addr ma.Multiaddr, pid peer.ID) (Conn, error)
	Listen(addr ma.Multiaddr) (Listener, error)
}

type Listener interface {
	Accept() (Conn, error)
	Close() error
	Multiaddr() ma.Multiaddr
}
