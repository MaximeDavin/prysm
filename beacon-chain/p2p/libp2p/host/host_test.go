package host_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/peer"
	test "github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/testing"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

// TODO(max): Most of the logic has been already tested in network_test
// Host acts like a proxy to Network, a good way of testing would be to have
// a network mock and test that Host calls the mock's methods.
func TestHostSimple(t *testing.T) {
	h1 := test.CreateHost(t)
	h2 := test.CreateHost(t)

	h2pinfo := &peer.AddrInfo{
		ID:    h2.Network().LocalPeer(),
		Addrs: h2.Network().Addrs(),
	}

	err := h1.Connect(context.Background(), *h2pinfo)
	require.NoError(t, err)
}
