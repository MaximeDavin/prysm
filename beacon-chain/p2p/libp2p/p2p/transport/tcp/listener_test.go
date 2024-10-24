package tcp

import (
	"context"
	"testing"
	"time"

	"libp2p/core/transport"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestListenerClose(t *testing.T) {
	tcp, pid := createTransport(t)
	l := createListener(t, tcp)

	errCh := make(chan error)
	defer close(errCh)
	go func() {
		_, err := l.Accept()
		errCh <- err
	}()

	select {
	case err := <-errCh:
		t.Fatalf("connection closed earlier than expected. expected nothing on channel, got: %v", err)
	case <-time.After(200 * time.Millisecond):
		// nothing in 200ms.
	}

	// unblocks Accept when it is closed.
	require.NoError(t, l.Close())
	require.ErrorContains(t, "use of closed network connection", <-errCh)

	// doesn't accept new connections when it is closed
	_, err := tcp.Dial(context.Background(), l.Multiaddr(), pid)
	require.ErrorContains(t, "refused", err)
}

func TestListenerCloseClosesQueued(t *testing.T) {
	tcp, pid := createTransport(t)
	l := createListener(t, tcp)

	var conns []transport.UpgradedConn
	for i := 0; i < 10; i++ {
		conn, err := tcp.Dial(context.Background(), l.Multiaddr(), pid)
		require.NoError(t, err)
		conns = append(conns, conn)
	}

	// wait for all the dials to happen.
	time.Sleep(100 * time.Millisecond)

	// all the connections are opened.
	for _, c := range conns {
		require.Equal(t, false, c.IsClosed())
	}

	// expect that all the connections will be closed.
	err := l.Close()
	require.NoError(t, err)

	for _, c := range conns {
		_ = c.Close()
	}
	// wait for all the connections to close.
	time.Sleep(500 * time.Millisecond)
	for _, c := range conns {
		require.Equal(t, true, c.IsClosed())
	}
}

func TestFailedUpgradeError(t *testing.T) {
	options := DefaultTcpTransportOptions
	options.StreamsSupported = "mockProtocol"

	ltcp, pid := createTransportWithOptions(t, options)
	l := createListener(t, ltcp)
	defer l.Close()

	errCh := make(chan error)
	go func() {
		_, err := l.Accept()
		errCh <- err
	}()

	dtcp, _ := createTransportWithOptions(t, options)
	dtcp.Dial(context.Background(), l.Multiaddr(), pid)

	require.ErrorContains(t, "stream multiplexing protocol not supported", <-errCh)

}
