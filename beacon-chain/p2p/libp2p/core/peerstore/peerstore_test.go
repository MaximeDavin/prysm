package peerstore

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func getMaddrs(t *testing.T, addrs ...string) []ma.Multiaddr {
	t.Helper()

	maddrs := make([]ma.Multiaddr, 0)
	for _, addr := range addrs {
		maddrs = append(maddrs, ma.StringCast(addr))
	}
	return maddrs
}

func getPeerInfo(t *testing.T, addrs ...string) peer.AddrInfo {
	t.Helper()

	maddrs := getMaddrs(t, addrs...)
	infos, err := peer.AddrInfosFromP2pAddrs(maddrs...)
	require.NoError(t, err)
	return infos[0]
}

const (
	onetcp  = "/ip4/192.168.0.1/tcp/5678"
	onequic = "/ip4/192.168.0.1/udp/5678/quic-v1"
	onep2p  = "/p2p/16Uiu2HAkum7hhuMpWqFj3yNLcmQBGmThmqw2ohaCRThXQuKU9ohs"
	twotcp  = "/ip4/192.168.0.2/tcp/5678"
	twoquic = "/ip4/192.168.0.2/udp/5678/quic-v1"
	twop2p  = "/p2p/QmUn6ycS8Fu6L462uZvuEfDoSgYX6kqP4aSZWMa7z1tWAX"
)

func TestSetAddrs(t *testing.T) {
	ps := NewPeerStore()
	require.Equal(t, 0, len(ps.addrs))

	// First let's add two address for this peer
	pinfo := getPeerInfo(t, onetcp+onep2p, onequic+onep2p)
	ps.SetAddrs(pinfo)
	// Check we have only one peer
	require.Equal(t, 1, len(ps.addrs))
	// Check that we have added the two addresses for this peer
	maddrs := getMaddrs(t, onetcp, onequic)
	require.DeepEqual(t, maddrs, ps.addrs[pinfo.ID])

	// Let's add a new address and check it has replaced the two previous ones for this peer
	newPinfo := getPeerInfo(t, twotcp+onep2p)
	ps.SetAddrs(newPinfo)
	// Check we still have only one peer
	require.Equal(t, 1, len(ps.addrs))
	// Check we have only the new address
	newAddr := getMaddrs(t, twotcp)
	require.DeepEqual(t, newAddr, ps.addrs[pinfo.ID])
}

func TestSetAddrsMultiplePeers(t *testing.T) {
	ps := NewPeerStore()
	require.Equal(t, 0, len(ps.addrs))

	// First let's add two peers
	onepinfo := getPeerInfo(t, onetcp+onep2p)
	ps.SetAddrs(onepinfo)
	twopinfo := getPeerInfo(t, twotcp+twop2p)
	ps.SetAddrs(twopinfo)

	// Check we have only two peers
	require.Equal(t, 2, len(ps.addrs))
}

func TestSetAddrsNoAddr(t *testing.T) {
	ps := NewPeerStore()
	require.Equal(t, 0, len(ps.addrs))

	// Let's add one peer with no address
	pinfo := peer.AddrInfo{ID: "id", Addrs: []ma.Multiaddr{}}
	ps.SetAddrs(pinfo)

	require.Equal(t, 1, len(ps.addrs))
	require.Equal(t, 0, len(ps.addrs[pinfo.ID]))
}

func TestSetAddrsRemoveDuplicates(t *testing.T) {
	ps := NewPeerStore()

	// Let's add duplicated address for this peer
	pinfo := getPeerInfo(t, onetcp+onep2p, onetcp+onep2p)
	ps.SetAddrs(pinfo)

	require.Equal(t, 1, len(ps.addrs))

	addr := getMaddrs(t, onetcp)
	require.DeepEqual(t, addr, ps.addrs[pinfo.ID])
}

func TestSetAddrsWrongAddrs(t *testing.T) {
	hook := logTest.NewGlobal()
	ps := NewPeerStore()
	// We add:
	// 1. a correct address
	pinfo := getPeerInfo(t, onetcp+onep2p)
	// 2. an address with only p2p part that will be discarded
	onlyP2PAddr := ma.StringCast(onep2p)
	pinfo.Addrs = append(pinfo.Addrs, onlyP2PAddr)
	// 3. an address with p2p part not matching the peer.AddrInfo.ID
	// that will be discarded
	wrongIdAddr := ma.StringCast(twotcp + twop2p)
	pinfo.Addrs = append(pinfo.Addrs, wrongIdAddr)

	ps.SetAddrs(pinfo)
	require.LogsContain(t, hook, "nil multiaddr")
	require.LogsContain(t, hook, "different peerId")
	require.Equal(t, 1, len(ps.addrs))
	require.DeepEqual(t, getMaddrs(t, onetcp), ps.addrs[pinfo.ID])
}

func TestAddrs(t *testing.T) {
	ps := NewPeerStore()

	// Let's add two address for this peer
	pinfo := getPeerInfo(t, onetcp+onep2p, onequic+onep2p)
	require.Equal(t, 0, len(ps.Addrs(pinfo.ID)))

	ps.SetAddrs(pinfo)

	maddrs := getMaddrs(t, onetcp, onequic)
	require.DeepEqual(t, maddrs, ps.Addrs(pinfo.ID))
	require.Equal(t, 0, len(ps.Addrs("unknown_peer")))
}
