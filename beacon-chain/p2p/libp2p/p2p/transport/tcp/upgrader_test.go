package tcp

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p/direction"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestUpgrade(t *testing.T) {
	testCases := []struct {
		desc   string
		opts   TcpTransportOptions
		trOpts TcpTransportOptions
		error  string
	}{
		{
			desc: "Success",
			opts: func() TcpTransportOptions {
				options := DefaultTcpTransportOptions
				options.SecuritySupported = "/noise"
				options.StreamsSupported = "/yamux/1.0.0"
				return options
			}(),
			trOpts: func() TcpTransportOptions {
				options := DefaultTcpTransportOptions
				options.SecuritySupported = "/noise"
				options.StreamsSupported = "/yamux/1.0.0"
				return options
			}(),
			error: "",
		},
		{
			desc: "No security protocol in common",
			opts: func() TcpTransportOptions {
				options := DefaultTcpTransportOptions
				options.SecuritySupported = "/invalidSecurity1/1.0"
				return options
			}(),
			trOpts: func() TcpTransportOptions {
				options := DefaultTcpTransportOptions
				options.SecuritySupported = "/invalidSecurity2/1.0"
				return options
			}(),
			error: "/invalidSecurity2/1.0",
		},
		{
			desc: "No stream protocol in common",
			opts: func() TcpTransportOptions {
				options := DefaultTcpTransportOptions
				options.StreamsSupported = "/invalidStream1/1.0"
				return options
			}(),
			trOpts: func() TcpTransportOptions {
				options := DefaultTcpTransportOptions
				options.StreamsSupported = "/invalidStream2/1.0"
				return options
			}(),
			error: "/invalidStream2/1.0",
		},
		{
			desc: "Security protocol in common, but not supported (not noise)",
			opts: func() TcpTransportOptions {
				options := DefaultTcpTransportOptions
				options.SecuritySupported = "/invalidSecurity1/1.0"
				return options
			}(),
			trOpts: func() TcpTransportOptions {
				options := DefaultTcpTransportOptions
				options.SecuritySupported = "/invalidSecurity1/1.0"
				return options
			}(),
			error: "security protocol not supported",
		},
		{
			desc: "Stream protocol in common, but not supported (not yamux)",
			opts: func() TcpTransportOptions {
				options := DefaultTcpTransportOptions
				options.StreamsSupported = "/mplex/1.0.0"
				return options
			}(),
			trOpts: func() TcpTransportOptions {
				options := DefaultTcpTransportOptions
				options.StreamsSupported = "/mplex/1.0.0"
				return options
			}(),
			error: "stream multiplexing protocol not supported",
		},
		{
			desc: "Timeout",
			trOpts: func() TcpTransportOptions {
				options := DefaultTcpTransportOptions
				options.AcceptTimeout = 0
				return options
			}(),
			error: "timeout",
		},
	}
	for _, tC := range testCases {
		t.Run("Outbound "+tC.desc, func(t *testing.T) {
			tcp, pid := createTransportWithOptions(t, tC.opts)
			l := createListener(t, tcp)
			defer l.Close()

			conn, err := manet.Dial(l.Multiaddr())
			require.NoError(t, err)

			tr, _ := createTransportWithOptions(t, tC.trOpts)
			_, err = Upgrade(context.Background(), tr, conn, pid, direction.DirOutbound)
			if tC.error != "" {
				require.ErrorContains(t, tC.error, err)
			} else {
				require.NoError(t, err)
			}
		})
		t.Run("Inbound "+tC.desc, func(t *testing.T) {
			addr, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
			l, _ := manet.Listen(addr)
			defer l.Close()

			ltcp, pid := createTransportWithOptions(t, tC.trOpts)
			dtcp, _ := createTransportWithOptions(t, tC.opts)
			go func() {
				dtcp.Dial(context.Background(), l.Multiaddr(), pid)
			}()

			conn, err := l.Accept()
			require.NoError(t, err)

			_, err = Upgrade(context.Background(), ltcp, conn, "", direction.DirInbound)
			if tC.error != "" {
				require.ErrorContains(t, tC.error, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Test that SetDeadline error is correctly returned
func TestSetDeadlineError(t *testing.T) {
	tcp, _ := createTransport(t)
	l := createListener(t, tcp)
	defer l.Close()

	conn, err := manet.Dial(l.Multiaddr())
	require.NoError(t, err)

	tr, _ := createTransport(t)
	// We close the connection so SetDeadline fails
	conn.Close()
	_, err = Upgrade(context.Background(), tr, conn, "", direction.DirInbound)
	require.ErrorContains(t, "closed", err)
}
