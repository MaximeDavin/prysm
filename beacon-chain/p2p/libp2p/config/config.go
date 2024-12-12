package config

import (
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	ma "github.com/multiformats/go-multiaddr"
)

// AddrsFactory is a function that takes a set of multiaddrs we're listening on and
// returns the set of multiaddrs we should advertise to the network.
type AddrsFactory = func(addrs []ma.Multiaddr) []ma.Multiaddr

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
	AddrsFactory AddrsFactory

	UseQuic bool

	Muxers []Muxer

	DialTimeout time.Duration
}

type Option func(cfg *Config) error

type Muxer struct {
	ID string
}

func NewConfig() *Config {
	return &Config{
		UseQuic:     false,
		DialTimeout: time.Second * 10,
	}
}

// Apply applies the given options to the config, returning the first error
// encountered (if any).
func (cfg *Config) Apply(opts ...Option) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(cfg); err != nil {
			return err
		}
	}
	return nil
}
