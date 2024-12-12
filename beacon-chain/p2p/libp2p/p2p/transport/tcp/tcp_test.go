package tcp

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/config"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/transport"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func createConfig(t *testing.T) *config.Config {
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

func createTransport(t *testing.T) (*TcpTransport, peer.ID) {
	t.Helper()

	cfg := createConfig(t)
	tcp := NewTCPTransport(cfg)
	return tcp, cfg.PeerId
}

func createTransportWithOptions(t *testing.T, options TcpTransportOptions) (*TcpTransport, peer.ID) {
	t.Helper()

	cfg := createConfig(t)
	tcp := NewTCPTransportWithOptions(cfg, options)
	return tcp, cfg.PeerId
}

func createListener(t *testing.T, tcp *TcpTransport) transport.Listener {
	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
	require.NoError(t, err)

	l, err := tcp.Listen(addr)
	require.NoError(t, err)
	return l
}

func testConn(t *testing.T, clientConn, serverConn transport.UpgradedConn) {
	t.Helper()

	cstr, err := clientConn.OpenStream(context.Background())
	require.NoError(t, err)

	_, err = cstr.Write([]byte("foobar"))
	require.NoError(t, err)

	sstr, err := serverConn.AcceptStream()
	require.NoError(t, err)

	b := make([]byte, 6)
	_, err = sstr.Read(b)
	require.NoError(t, err)
	require.DeepEqual(t, []byte("foobar"), b)

	_, err = sstr.Write([]byte("pong"))
	require.NoError(t, err)
	c := make([]byte, 4)
	_, err = cstr.Read(c)
	require.NoError(t, err)
	require.DeepEqual(t, []byte("pong"), c)
}

func TestTCP(t *testing.T) {
	ltcp, pid := createTransport(t)
	l := createListener(t, ltcp)
	defer l.Close()

	dtcp, _ := createTransport(t)

	t.Run("accept single conn", func(t *testing.T) {
		cconn, err := dtcp.Dial(context.Background(), l.Multiaddr(), pid)
		require.NoError(t, err)

		sconn, err := l.Accept()
		require.NoError(t, err)

		testConn(t, cconn, sconn)
	})
	t.Run("accept multiple conns", func(t *testing.T) {
		var toClose []io.Closer
		defer func() {
			for _, c := range toClose {
				_ = c.Close()
			}
		}()

		var ctx = context.Background()
		for i := 0; i < 10; i++ {
			cconn, err := dtcp.Dial(ctx, l.Multiaddr(), pid)
			require.NoError(t, err)
			toClose = append(toClose, cconn)

			sconn, err := l.Accept()
			require.NoError(t, err)
			toClose = append(toClose, sconn)

			testConn(t, cconn, sconn)
		}
	})
}

func TestConnectionsClosedIfNotAccepted(t *testing.T) {
	var timeout = 100 * time.Millisecond
	options := DefaultTcpTransportOptions
	options.AcceptTimeout = timeout

	ltcp, pid := createTransportWithOptions(t, options)
	l := createListener(t, ltcp)
	defer l.Close()

	var ctx = context.Background()
	dtcp, _ := createTransport(t)
	conn, err := dtcp.Dial(ctx, l.Multiaddr(), pid)
	require.NoError(t, err)

	errCh := make(chan error)
	go func() {
		defer conn.Close()
		str, err := conn.OpenStream(context.Background())
		if err != nil {
			errCh <- err
			return
		}
		// start a Read. It will block until the connection is closed
		_, _ = str.Read([]byte{0})
		errCh <- nil
	}()

	time.Sleep(timeout / 2)
	select {
	case err := <-errCh:
		t.Fatalf("connection closed earlier than expected. expected nothing on channel, got: %v", err)
	default:
	}

	time.Sleep(timeout)
	require.NoError(t, <-errCh)
}

// Check that upgrade error is correctly returned to the dialer
func TestFailedUpgrade(t *testing.T) {
	ltcp, pid := createTransport(t)
	l := createListener(t, ltcp)
	defer l.Close()

	options := DefaultTcpTransportOptions
	options.StreamsSupported = "testErrorMuxer"
	dtcp, _ := createTransportWithOptions(t, options)

	_, err := dtcp.Dial(context.Background(), l.Multiaddr(), pid)
	require.ErrorContains(t, "protocols not supported: [testErrorMuxer]", err)
}

// Check that the manet.Listen is correctly returned
func TestListenError(t *testing.T) {
	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/udp/0")
	require.NoError(t, err)

	tcp, _ := createTransport(t)
	_, err = tcp.Listen(addr)

	require.ErrorContains(t, "unexpected address type", err)
}
