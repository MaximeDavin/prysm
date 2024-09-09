package peer

import (
	ma "github.com/multiformats/go-multiaddr"
)

type ID string

// AddrInfo is a small struct used to pass around a peer with
// a set of addresses.
type AddrInfo struct {
	ID    ID
	Addrs []ma.Multiaddr
}

// SplitAddr splits a p2p Multiaddr into a transport multiaddr and a peer ID.
//
// * Returns a nil transport if the address only contains a /p2p part.
// * Returns an empty peer ID if the address doesn't contain a /p2p part.
func SplitAddr(m ma.Multiaddr) (transport ma.Multiaddr, id ID) {
	if m == nil {
		return nil, ""
	}

	transport, p2ppart := ma.SplitLast(m)
	if p2ppart == nil || p2ppart.Protocol().Code != ma.P_P2P {
		return m, ""
	}
	id = ID(p2ppart.RawValue()) // already validated by the multiaddr library.
	return transport, id
}
