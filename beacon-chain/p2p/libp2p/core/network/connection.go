package network

import (
	"libp2p/core/transport"
)

type Conn struct {
	conn    transport.UpgradedConn
	network *Network
	stat    Stats
}

func NewConn(conn transport.UpgradedConn, n *Network, direction transport.Direction) *Conn {
	c := &Conn{
		conn:    conn,
		network: n,
		stat:    Stats{Direction: direction},
	}
	return c
}

func (c *Conn) Close() error {
	return c.conn.Close()
}
