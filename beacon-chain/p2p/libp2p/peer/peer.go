package peer

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/crypto"

	b58 "github.com/mr-tron/base58/base58"
	ma "github.com/multiformats/go-multiaddr"
	mh "github.com/multiformats/go-multihash"
)

type ID string

func (id ID) String() string {
	return b58.Encode([]byte(id))
}

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

// IDFromPublicKey returns the Peer ID corresponding to the public key pk.
func IDFromPublicKey(pk crypto.PubKey) (ID, error) {
	b, err := crypto.MarshalPublicKey(pk)
	if err != nil {
		return "", err
	}
	hash, _ := mh.Sum(b, mh.IDENTITY, -1)
	return ID(hash), nil
}
