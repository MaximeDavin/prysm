package noise

import (
	"fmt"
	"net"

	"libp2p/core/peer"
	"libp2p/core/transport"
)

type SecureConn interface {
	net.Conn
	transport.SecureConnMixin
}

type ErrPeerIDMismatch struct {
	Expected peer.ID
	Actual   peer.ID
}

func (e ErrPeerIDMismatch) Error() string {
	return fmt.Sprintf("peer id mismatch: expected %s, but remote key matches %s", e.Expected, e.Actual)
}
