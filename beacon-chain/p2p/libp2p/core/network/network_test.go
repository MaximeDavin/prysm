package network_test

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"libp2p"
	"libp2p/core/crypto"
	"libp2p/core/host"
	"libp2p/core/network"
	"libp2p/core/peer"
	"libp2p/core/protocol"
	"libp2p/core/transport"
	tests_utils "libp2p/p2p/testing"
	"libp2p/p2p/transport/tcp"

	"github.com/prysmaticlabs/prysm/v5/testing/require"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

const SLEEP_TIME = 50 * time.Millisecond

func createHosts(t *testing.T) (*host.Host, *host.Host, *peer.AddrInfo, *peer.AddrInfo) {
	t.Helper()

	h1 := tests_utils.CreateHost(t)

	h2 := tests_utils.CreateHost(t)

	n1pinfo := &peer.AddrInfo{
		ID:    h1.Network().LocalPeer(),
		Addrs: h1.Addrs(),
	}

	n2pinfo := &peer.AddrInfo{
		ID:    h2.Network().LocalPeer(),
		Addrs: h2.Addrs(),
	}

	return h1, h2, n1pinfo, n2pinfo
}

func TestNetworkSimple(t *testing.T) {
	h1, h2, _, h2pinfo := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	require.NoError(t, h1.Connect(context.Background(), *h2pinfo))

	piper, pipew := io.Pipe()
	h2.SetStreamHandler("test_proto", func(s transport.Stream) {
		defer s.Close()
		w := io.MultiWriter(s, pipew)
		io.Copy(w, s) // mirror everything
	})

	s, err := h1.NewStream(context.Background(), h2pinfo.ID, "test_proto")
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
	h1, h2, _, h2pinfo := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	// No connection to n2
	require.Equal(t, 0, len(h1.Network().ConnsToPeer(h2pinfo.ID)))
	require.NoError(t, h1.Connect(context.Background(), *h2pinfo))

	// There is one connection to n2
	require.NotNil(t, h1.Network().ConnsToPeer(h2pinfo.ID))
	require.Equal(t, 1, len(h1.Network().ConnsToPeer(h2pinfo.ID)))

	// We keep a pointer to the conn for later comparison
	conn := h1.Network().ConnsToPeer(h2pinfo.ID)[0]

	// Let's connect again and check if the same connection is reused
	require.NoError(t, h1.Connect(context.Background(), *h2pinfo))
	// The peer was not dialed, the previous connection was reused
	require.Equal(t, h1.Network().ConnsToPeer(h2pinfo.ID)[0], conn)
}

// After two consecutive Connect's calls, check that a new connection is established
// if the first connection has been closed.
func TestConnectClosedConnection(t *testing.T) {
	h1, h2, _, h2pinfo := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	// No connection to n2
	require.Equal(t, 0, len(h1.Network().ConnsToPeer(h2pinfo.ID)))
	require.NoError(t, h1.Connect(context.Background(), *h2pinfo))

	// There is one connection to n2
	require.NotNil(t, h1.Network().ConnsToPeer(h2pinfo.ID))
	require.Equal(t, 1, len(h1.Network().ConnsToPeer(h2pinfo.ID)))

	// We keep a pointer to the conn for later comparison
	conn := h1.Network().ConnsToPeer(h2pinfo.ID)[0]

	// We close the connection
	conn.Close()

	require.NoError(t, h1.Connect(context.Background(), *h2pinfo))

	// We should now have a new connection
	require.Equal(t, 1, len(h1.Network().ConnsToPeer(h2pinfo.ID)))
	require.NotEqual(t, h1.Network().ConnsToPeer(h2pinfo.ID)[0], conn)
}

func TestSimultDials(t *testing.T) {
	ctx := context.Background()
	h1, h2, h1pinfo, h2pinfo := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	// connect everyone
	{
		var wg sync.WaitGroup
		errs := make(chan error, 20) // 2 connect calls in each of the 10 for-loop iterations

		logrus.Info("Connecting swarms simultaneously.")
		for i := 0; i < 10; i++ { // connect 10x for each.
			wg.Add(2)
			go func() {
				logrus.Infof("Loop %v :connecting n1 to n2...", i)
				h1.Connect(ctx, *h2pinfo)
				wg.Done()
			}()
			go func() {
				logrus.Infof("Loop %v :connecting n2 to n1...", i)
				h2.Connect(ctx, *h1pinfo)
				wg.Done()
			}()
		}
		wg.Wait()
		close(errs)

		for err := range errs {
			require.NoError(t, err)
			if err != nil {
				t.Fatal("error swarm dialing to peer", err)
			}
		}
	}

}

// Check that peer's addresses are added to the peerstore when Connect
// is called
func TestConnectAbsorbPeer(t *testing.T) {
	h1, h2, _, h2pinfo := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	// n2pinfo.Addrs = []ma.Multiaddr{n2.tcpListener.Multiaddr()}
	require.NoError(t, h1.Connect(context.Background(), *h2pinfo))
	require.DeepEqual(t, h2pinfo.Addrs, h1.Peerstore().Addrs(h2pinfo.ID))
}

// Check that the DialPeer error bubbles up correctly
func TestConnectErrorDialFail(t *testing.T) {
	h1, h2, _, h2pinfo := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	// Try to dial an address that n2 does not listen to
	addr, _ := ma.NewMultiaddr("/ip4/127.0.0.2/tcp/1")
	h2pinfo.Addrs = []ma.Multiaddr{addr}
	require.ErrorContains(t, "Failed to dial", h1.Connect(context.Background(), *h2pinfo))
}

// Check DialPeer error when there is no addresses for this peer in the peerstore
func TestDialPeerNoAddresses(t *testing.T) {
	h1, h2, h1pinfo, _ := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	_, err := h1.Network().DialPeer(context.Background(), h1pinfo.ID)
	require.ErrorContains(t, "has no addresses", err)
}

// Check that DialPeer log every failed dial attempt and return an error when no
// there is no more addresses to dial
func TestDialPeerAllAddressesFail(t *testing.T) {
	hook := logTest.NewGlobal()
	h1, h2, _, h2pinfo := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	// We add two addresses to n2
	addr1, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/1")
	addr2, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/2")
	h2pinfo.Addrs = []ma.Multiaddr{addr1, addr2}
	h1.Peerstore().SetAddrs(*h2pinfo)

	_, err := h1.Network().DialPeer(context.Background(), h2pinfo.ID)
	require.LogsContain(t, hook, "/ip4/127.0.0.1/tcp/1")
	require.LogsContain(t, hook, "/ip4/127.0.0.1/tcp/2")
	require.ErrorContains(t, "Failed to dial peer", err)
}

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
		// {
		// 	Name:     "valid ip4 + quic",
		// 	Addr:     "/ip4/127.0.0.1/udp/1/quic-v1",
		// 	Expected: quic.ID,
		// },
		// {
		// 	Name:     "valid ip6 + quic",
		// 	Addr:     "/ip6/::1/udp/1/quic-v1",
		// 	Expected: quic.ID,
		// },
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
			Error: "invalid",
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
			id, err := network.GetTransportID(addr)
			if tc.Error != "" {
				require.ErrorContains(t, tc.Error, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.Expected, id)
			}
		})
	}
}

// Verify that Close can be call multiple time without error
func TestMultipleClose(t *testing.T) {
	h := tests_utils.CreateHost(t)
	require.NoError(t, h.Close())
	require.NoError(t, h.Close())
	require.NoError(t, h.Close())
}

func TestClosePeer(t *testing.T) {
	ctx := context.Background()

	h1, h2, h1pinfo, h2pinfo := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	require.NoError(t, h2.Connect(ctx, *h1pinfo))
	require.NoError(t, h1.Connect(ctx, *h2pinfo))

	require.Equal(t, 2, len(h1.Network().ConnsToPeer(h2pinfo.ID)))
	require.NoError(t, h1.Network().ClosePeer(h2pinfo.ID))
	time.Sleep(SLEEP_TIME)
	require.Equal(t, 0, len(h1.Network().ConnsToPeer(h2pinfo.ID)))
}

func TestListenAddresses(t *testing.T) {
	h := tests_utils.CreateHost(t)
	addrs := h.Network().ListenAddresses()
	require.Equal(t, 1, len(addrs))
	require.StringContains(t, "/ip4/127.0.0.1/tcp/", addrs[0].String())
}

func TestConns(t *testing.T) {
	ctx := context.Background()

	h1, h2, h1pinfo, h2pinfo := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	h3, h4, h3pinfo, h4pinfo := createHosts(t)
	defer h3.Close()
	defer h4.Close()

	require.Equal(t, 0, len(h1.Network().Conns()))

	require.NoError(t, h1.Connect(ctx, *h2pinfo))
	require.NoError(t, h1.Connect(ctx, *h3pinfo))
	require.NoError(t, h1.Connect(ctx, *h4pinfo))

	// Let's wait for connections to complete
	time.Sleep(SLEEP_TIME)

	require.Equal(t, 3, len(h1.Network().Conns()))

	require.NoError(t, h2.Connect(ctx, *h1pinfo))
	require.NoError(t, h3.Connect(ctx, *h1pinfo))
	require.NoError(t, h4.Connect(ctx, *h1pinfo))

	// Let's wait for connections to complete
	time.Sleep(SLEEP_TIME)
	require.Equal(t, 6, len(h1.Network().Conns()))
}

func TestConnsToPeer(t *testing.T) {
	ctx := context.Background()

	h1, h2, h1pinfo, h2pinfo := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	require.Equal(t, 0, len(h1.Network().ConnsToPeer(h2pinfo.ID)))

	require.NoError(t, h1.Connect(ctx, *h2pinfo))
	time.Sleep(SLEEP_TIME)
	require.Equal(t, 1, len(h1.Network().ConnsToPeer(h2pinfo.ID)))

	require.NoError(t, h2.Connect(ctx, *h1pinfo))
	time.Sleep(SLEEP_TIME)
	require.Equal(t, 2, len(h1.Network().ConnsToPeer(h2pinfo.ID)))
}

func TestPeers(t *testing.T) {
	ctx := context.Background()

	h1, h2, _, h2pinfo := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	h3, h4, h3pinfo, h4pinfo := createHosts(t)
	defer h3.Close()
	defer h4.Close()

	require.Equal(t, 0, len(h1.Network().Peers()))

	require.NoError(t, h1.Connect(ctx, *h2pinfo))
	require.NoError(t, h1.Connect(ctx, *h3pinfo))
	require.NoError(t, h1.Connect(ctx, *h4pinfo))

	require.Equal(t, 3, len(h1.Network().Peers()))
}

func TestNotify(t *testing.T) {
	h1, h2, h1pinfo, h2pinfo := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	done := make(chan any, 2)
	h1.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, conn network.Conn) {
			done <- struct{}{}
		},
	})

	// Test outbound connection
	require.NoError(t, h1.Connect(context.Background(), *h2pinfo))
	// Test inbound connection
	require.NoError(t, h2.Connect(context.Background(), *h1pinfo))

	<-done
	<-done
}

func TestDialTimeout(t *testing.T) {
	hook := logTest.NewGlobal()
	h1 := tests_utils.CreateHostWithTimeout(t, time.Nanosecond)
	defer h1.Close()

	h2 := tests_utils.CreateHost(t)
	defer h2.Close()
	h2pinfo := peer.AddrInfo{
		ID:    h2.Network().LocalPeer(),
		Addrs: h2.Network().ListenAddresses(),
	}

	err := h1.Connect(context.Background(), h2pinfo)
	time.Sleep(SLEEP_TIME)
	require.LogsContain(t, hook, "deadline exceeded")
	require.ErrorContains(t, "Failed to dial", err)
}

// Check that we can remove a stream handler
func TestRemoveStreamHandler(t *testing.T) {
	h1, h2, _, h2pinfo := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	require.NoError(t, h1.Connect(context.Background(), *h2pinfo))

	h2.SetStreamHandler("test1", func(s transport.Stream) {})
	h2.SetStreamHandler("test2", func(s transport.Stream) {})
	require.DeepEqual(t, []protocol.ID{"test1", "test2"}, h2.Mux().Protocols())

	h2.RemoveStreamHandler("test2")
	require.DeepEqual(t, []protocol.ID{"test1"}, h2.Mux().Protocols())
}

func TestWrongListeningAddr(t *testing.T) {
	privk, _, err := crypto.GenerateSecp256k1Key()
	require.NoError(t, err)

	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/1")
	require.NoError(t, err)

	opts := []libp2p.Option{
		libp2p.Identity(privk),
		libp2p.ListenAddrs(addr),
	}
	_, err = libp2p.New(opts...)
	require.NoError(t, err)

	// Let's reuse the same listening address to produce an error
	_, err = libp2p.New(opts...)
	require.ErrorContains(t, "127.0.0.1:1", err)
}

func TestNewStreamDialError(t *testing.T) {
	hook := logTest.NewGlobal()
	h1, h2, _, h2pinfo := createHosts(t)
	defer h1.Close()

	require.NoError(t, h1.Connect(context.Background(), *h2pinfo))

	h2.Close()
	time.Sleep(SLEEP_TIME)

	_, err := h1.NewStream(context.Background(), h2pinfo.ID, "test_proto")
	require.ErrorContains(t, "Failed to dial", err)
	require.LogsContain(t, hook, "Failed to dial peer")
}

func TestNewStreamUnsupportedProtocolError(t *testing.T) {
	h1, h2, _, h2pinfo := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	require.NoError(t, h1.Connect(context.Background(), *h2pinfo))

	h2.SetStreamHandler("test_proto", func(s transport.Stream) {})

	_, err := h1.NewStream(context.Background(), h2pinfo.ID, "unknown_proto")
	require.ErrorContains(t, "unknown_proto", err)
}

func TestNewStreamDialWhenNoConnection(t *testing.T) {
	h1, h2, _, h2pinfo := createHosts(t)
	defer h1.Close()
	defer h2.Close()

	require.NoError(t, h1.Connect(context.Background(), *h2pinfo))
	require.NoError(t, h1.Network().ClosePeer(h2pinfo.ID))
	time.Sleep(SLEEP_TIME)

	h2.SetStreamHandler("test_proto", func(s transport.Stream) {})

	_, err := h1.NewStream(context.Background(), h2pinfo.ID, "test_proto")
	require.NoError(t, err)
}
