package tests

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/config"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/prysmaticlabs/prysm/v5/testing/require"

	ma "github.com/multiformats/go-multiaddr"
)

func CreateHost(t *testing.T) host.Host {
	t.Helper()
	h, err := libp2p.New(createOptions(t, time.Second*5)...)
	require.NoError(t, err)
	return h
}

func CreateHostWithTimeout(t *testing.T, timeout time.Duration) host.Host {
	h, err := libp2p.New(createOptions(t, timeout)...)

	require.NoError(t, err)
	return h
}

func createOptions(t *testing.T, timeout time.Duration) []libp2p.Option {
	privk, _, err := crypto.GenerateSecp256k1Key()
	require.NoError(t, err)

	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
	require.NoError(t, err)

	return []libp2p.Option{
		libp2p.Identity(privk),
		libp2p.ListenAddrs(addr),
		libp2p.Muxer("", ""),
		libp2p.DialTimeout(timeout),
	}
}

func CreateConfig(t *testing.T) *config.Config {
	t.Helper()

	cfg := config.NewConfig()

	privk, _, err := crypto.GenerateSecp256k1Key()
	require.NoError(t, err)
	cfg.PeerKey = privk

	pid, err := peer.IDFromPrivateKey(privk)
	require.NoError(t, err)
	cfg.PeerId = pid

	return cfg
}
