package network

import (
	"github.com/libp2p/go-libp2p/core/transport"
	"github.com/libp2p/go-libp2p/direction"
)

type Conn transport.UpgradedConn

type NetworkConn struct {
	conn    transport.UpgradedConn
	network *Network
	stat    Stats
}

func NewConn(conn transport.UpgradedConn, n *Network, direction direction.Direction) *NetworkConn {
	c := &NetworkConn{
		conn:    conn,
		network: n,
		stat:    Stats{Direction: direction},
	}
	return c
}

func (c *NetworkConn) Close() error {
	return c.conn.Close()
}

// DEPRECACTED Use `github.com/libp2p/go-libp2p/transport.ConnMultiaddrs` instead
type ConnMultiaddrs transport.ConnMultiaddrs
