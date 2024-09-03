package network

import (
	"context"
	"io"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/peer"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/quic"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/tcp"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/transport"
	"github.com/prysmaticlabs/prysm/v5/testing/require"

	ma "github.com/multiformats/go-multiaddr"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func createNetworks(t *testing.T) (*network, *network, *peer.AddrInfo, *peer.AddrInfo) {
	t.Helper()

	n1 := NewNetwork()
	defer n1.Close()

	n2 := NewNetwork()
	defer n2.Close()

	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
	require.NoError(t, err)

	n1pinfo := &peer.AddrInfo{
		ID:    "n1",
		Addrs: []ma.Multiaddr{addr},
	}

	n2pinfo := &peer.AddrInfo{
		ID:    "n2",
		Addrs: []ma.Multiaddr{addr},
	}

	return n1, n2, n1pinfo, n2pinfo
}

func TestNetworkSimple(t *testing.T) {
	n1, n2, n1pinfo, n2pinfo := createNetworks(t)
	n2.Listen(n1pinfo.Addrs...)
	defer n2.Close()

	n2pinfo.Addrs = []ma.Multiaddr{n2.tcpListener.Multiaddr()}
	require.NoError(t, n1.Connect(context.Background(), *n2pinfo))

	piper, pipew := io.Pipe()
	n2.SetStreamHandler("test_proto", func(s transport.Stream) {
		defer s.Close()
		w := io.MultiWriter(s, pipew)
		io.Copy(w, s) // mirror everything
	})

	s, err := n1.NewStream(context.Background(), n2pinfo.ID, "test_proto")
	require.NoError(t, err)

	// write to the stream
	buf1 := []byte("abcdefghijkl")
	_, err = s.Write(buf1)
	require.NoError(t, err)

	// get it from the stream (echoed)
	buf2 := make([]byte, len(buf1))
	_, err = s.Read(buf2)
	// _, err = io.ReadFull(s, buf2)
	require.NoError(t, err)
	require.DeepEqual(t, buf1, buf2)

	// get it from the pipe (tee)
	buf3 := make([]byte, len(buf1))
	_, err = io.ReadFull(piper, buf3)
	require.NoError(t, err)
	require.DeepEqual(t, buf1, buf3)
}

// Check that two consecutive Connect's calls reuse the connection
// and that only one dial is done
func TestConnectReuseConnection(t *testing.T) {
	n1, n2, n1pinfo, n2pinfo := createNetworks(t)
	n2.Listen(n1pinfo.Addrs...)
	defer n2.Close()

	// No connection to n2
	require.IsNil(t, n1.conns[n2pinfo.ID])
	n2pinfo.Addrs = []ma.Multiaddr{n2.tcpListener.Multiaddr()}
	require.NoError(t, n1.Connect(context.Background(), *n2pinfo))

	// There is one connection to n2
	require.NotNil(t, n1.conns[n2pinfo.ID])

	// We keep a pointer to the conn for later comparison
	conn := n1.conns[n2pinfo.ID]

	// Removing n2 address guarantees that DialPeer will fail. If connect return
	// no error, it means that DialPeer was not called
	n2pinfo.Addrs = []ma.Multiaddr{}
	require.NoError(t, n1.Connect(context.Background(), *n2pinfo))

	// The  peer was not dialed, the previous connection was reused
	require.Equal(t, n1.conns[n2pinfo.ID], conn)
}

// Check that peer's addresses are added to the peerstore when Connect
// is called
func TestConnectAbsorbPeer(t *testing.T) {
	n1, n2, n1pinfo, n2pinfo := createNetworks(t)
	n2.Listen(n1pinfo.Addrs...)
	defer n2.Close()

	n2pinfo.Addrs = []ma.Multiaddr{n2.tcpListener.Multiaddr()}
	require.NoError(t, n1.Connect(context.Background(), *n2pinfo))
	require.DeepEqual(t, n2pinfo.Addrs, n1.peerstore.Addrs(n2pinfo.ID))
}

// Check that the DialPeer error bubbles up correctly
func TestConnectErrorDialFail(t *testing.T) {
	n1, _, _, n2pinfo := createNetworks(t)
	require.ErrorContains(t, "Failed to dial", n1.Connect(context.Background(), *n2pinfo))
}

// Check DialPeer error when there is no addresses for this peer in the peerstore
func TestDialPeerNoAddresses(t *testing.T) {
	n1, _, n1pinfo, _ := createNetworks(t)
	_, err := n1.DialPeer(context.Background(), n1pinfo.ID)
	require.ErrorContains(t, "has no addresses", err)
}

// Check that DialPeer log every failed dial attempt and return an error when no
// there is no more addresses to dial
func TestDialPeerAllAddressesFail(t *testing.T) {
	hook := logTest.NewGlobal()
	n1, _, _, n2pinfo := createNetworks(t)

	// We add two addresses to n2
	addr1, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/1")
	addr2, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/2")
	n2pinfo.Addrs = []ma.Multiaddr{addr1, addr2}
	n1.peerstore.SetAddrs(*n2pinfo)

	_, err := n1.DialPeer(context.Background(), n2pinfo.ID)
	require.LogsContain(t, hook, "/ip4/127.0.0.1/tcp/1")
	require.LogsContain(t, hook, "/ip4/127.0.0.1/tcp/2")
	require.ErrorContains(t, "Failed to dial peer", err)
}

// Check that DialPeer can call a ip4 tcp address
func TestDialPeerTCPDialer(t *testing.T) {
	n1, n2, _, n2pinfo := createNetworks(t)
	n2.Listen(n2pinfo.Addrs...)
	n2.Close()

	n2pinfo.Addrs = []ma.Multiaddr{n2.tcpListener.Multiaddr()}
	n1.peerstore.SetAddrs(*n2pinfo)
	_, err := n1.DialPeer(context.Background(), n2pinfo.ID)
	require.NoError(t, err)
}

// TODO(quic): Check that DialPeer can call a ip4 quic address
func TestDialPeerQuicDialer(t *testing.T) {}

func TestGetTransportID(t *testing.T) {
	type testcase struct {
		Name     string
		Addr     string
		Expected string
		Error    string
	}
	testcases := []testcase{
		{
			Name:     "valid ip4 + tcp",
			Addr:     "/ip4/127.0.0.1/tcp/1",
			Expected: tcp.ID,
		},
		{
			Name:     "valid ip6 + tcp",
			Addr:     "/ip6/::1/tcp/1",
			Expected: tcp.ID,
		},
		{
			Name:     "valid ip4 + quic",
			Addr:     "/ip4/127.0.0.1/udp/1/quic-v1",
			Expected: quic.ID,
		},
		{
			Name:     "valid ip6 + quic",
			Addr:     "/ip6/::1/udp/1/quic-v1",
			Expected: quic.ID,
		},
		{
			Name:  "not IP",
			Addr:  "/dns4/example.com/tcp/443",
			Error: "neither IP4 nor IP6",
		},
		{
			Name:  "not TCP nor UDP",
			Addr:  "/ip4/127.0.0.1/dns/443",
			Error: "neither TCP nor UDP",
		},
		{
			Name:  "missing transport",
			Addr:  "/ip4/127.0.0.1",
			Error: "neither TCP nor UDP",
		},
		{
			Name:  "UDP without quic",
			Addr:  "/ip4/127.0.0.1/udp/443",
			Error: "does not support quic-v1",
		},
		{
			Name:     "extra protocols",
			Addr:     "/ip4/127.0.0.1/tcp/443/tls/sni/example.com",
			Expected: tcp.ID,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			addr, err := ma.NewMultiaddr(tc.Addr)
			require.NoError(t, err)
			id, err := getTransportID(addr)
			if tc.Error != "" {
				require.ErrorContains(t, tc.Error, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.Expected, id)
			}
		})
	}
}
