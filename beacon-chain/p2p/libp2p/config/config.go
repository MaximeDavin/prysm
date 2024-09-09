package config

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/crypto"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/peer"

	ma "github.com/multiformats/go-multiaddr"
)

// Config describes a set of settings for a libp2p node
//
// This is *not* a stable interface. Use the options defined in the root
// package.
type Config struct {
	PeerKey crypto.PrivKey
	PeerId  peer.ID
	// UserAgent is the identifier this node will send to other peers when
	// identifying itself, e.g. via the identify protocol.
	UserAgent string

	ListenAddrs  []ma.Multiaddr
	AddrsFactory func(addrs []ma.Multiaddr) []ma.Multiaddr
}

func NewConfig() *Config {
	return &Config{}
}
