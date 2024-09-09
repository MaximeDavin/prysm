package testing_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/crypto"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/host"
	"github.com/prysmaticlabs/prysm/v5/testing/require"

	ma "github.com/multiformats/go-multiaddr"
)

func CreateHost(t *testing.T) host.Host {
	t.Helper()
	h, err := libp2p.New(createOptions(t)...)
	require.NoError(t, err)
	return h
}

func createOptions(t *testing.T) []libp2p.Option {
	privk, _, err := crypto.GenerateSecp256k1Key()
	require.NoError(t, err)

	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
	require.NoError(t, err)

	return []libp2p.Option{
		libp2p.Identity(privk),
		libp2p.ListenAddrs(addr),
	}
}
