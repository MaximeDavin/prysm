package peer

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/libp2p/go-libp2p/core/crypto"

	b58 "github.com/mr-tron/base58/base58"
	ma "github.com/multiformats/go-multiaddr"
	mh "github.com/multiformats/go-multihash"
)

type ID string

func (id ID) String() string {
	return b58.Encode([]byte(id))
}

// MarshalBinary returns the byte representation of the peer ID.
func (id ID) MarshalBinary() ([]byte, error) {
	return []byte(id), nil
}

// AddrInfo is a small struct used to pass around a peer with
// a set of addresses.
type AddrInfo struct {
	ID    ID
	Addrs []ma.Multiaddr
}

func (pi AddrInfo) String() string {
	return fmt.Sprintf("{%v: %v}", pi.ID, pi.Addrs)
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

// IDFromPublicKey returns the Peer ID corresponding to the private key pk.
func IDFromPrivateKey(pk crypto.PrivKey) (ID, error) {
	return IDFromPublicKey(pk.GetPublic())
}

// IDFromBytes casts a byte slice to the ID type, and validates
// the value to make sure it is a multihash.
func IDFromBytes(b []byte) (ID, error) {
	if _, err := mh.Cast(b); err != nil {
		return ID(""), err
	}
	return ID(b), nil
}

var ErrInvalidAddr = fmt.Errorf("invalid p2p multiaddr")

// AddrInfoFromP2pAddr converts a Multiaddr to an AddrInfo.
func AddrInfoFromP2pAddr(m ma.Multiaddr) (*AddrInfo, error) {
	transport, id := SplitAddr(m)
	if id == "" {
		return nil, ErrInvalidAddr
	}
	info := &AddrInfo{ID: id}
	if transport != nil {
		info.Addrs = []ma.Multiaddr{transport}
	}
	return info, nil
}

// AddrInfosFromP2pAddrs converts a set of Multiaddrs to a set of AddrInfos.
func AddrInfosFromP2pAddrs(maddrs ...ma.Multiaddr) ([]AddrInfo, error) {
	m := make(map[ID][]ma.Multiaddr)
	for _, maddr := range maddrs {
		transport, id := SplitAddr(maddr)
		if id == "" {
			return nil, errors.Errorf("invalid p2p multiaddr")
		}
		if transport == nil {
			if _, ok := m[id]; !ok {
				m[id] = nil
			}
		} else {
			m[id] = append(m[id], transport)
		}
	}
	ais := make([]AddrInfo, 0, len(m))
	for id, maddrs := range m {
		ais = append(ais, AddrInfo{ID: id, Addrs: maddrs})
	}
	return ais, nil
}

// AddrInfoFromString builds an AddrInfo from the string representation of a Multiaddr
func AddrInfoFromString(s string) (*AddrInfo, error) {
	a, err := ma.NewMultiaddr(s)
	if err != nil {
		return nil, err
	}

	return AddrInfoFromP2pAddr(a)
}

// Decode accepts an encoded peer ID and returns the decoded ID if the input is
// valid.
func Decode(s string) (ID, error) {
	m, err := mh.FromB58String(s)
	if err != nil {
		return "", fmt.Errorf("failed to parse peer ID: %s", err)
	}
	return ID(m), nil

}
