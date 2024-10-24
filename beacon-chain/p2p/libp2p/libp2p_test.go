package libp2p_test

import (
	"testing"
	"time"

	"libp2p"
	"libp2p/core/crypto"
	"libp2p/core/network"
	"libp2p/p2p/security/noise"
	"libp2p/p2p/transport/tcp"

	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/testing/require"

	ma "github.com/multiformats/go-multiaddr"
)

func prysmBasicOptions(t *testing.T) []libp2p.Option {
	t.Helper()

	privk, _, err := crypto.GenerateSecp256k1Key()
	require.NoError(t, err)

	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
	require.NoError(t, err)

	opts := []libp2p.Option{
		libp2p.Identity(privk),
		libp2p.ListenAddrs(addr),
		libp2p.UserAgent(version.BuildData()),
		libp2p.ConnectionGater(nil),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.DefaultMuxers,
		// libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Ping(false),
		libp2p.DisableRelay(),
	}
	return opts
}

// TODO(max): Add quic, natport, relay, hostadrdress, dns
// func withQuic(t *testing.T) []libp2p.Option {
// 	opts := prysmBasicOptions(t)
// 	return opts
// }

func withNatPort(t *testing.T) []libp2p.Option {
	opts := prysmBasicOptions(t)
	opts = append(opts, libp2p.NATPortMap())
	return opts
}

func withResourceManager(t *testing.T) []libp2p.Option {
	opts := prysmBasicOptions(t)
	opts = append(opts, libp2p.ResourceManager(&network.NullResourceManager{}))
	return opts
}

func withAddrsFactory(t *testing.T) []libp2p.Option {
	opts := prysmBasicOptions(t)
	opts = append(opts, libp2p.AddrsFactory(func(addrs []ma.Multiaddr) []ma.Multiaddr {
		return addrs
	}))
	return opts
}

func withDialTimeout(t *testing.T) []libp2p.Option {
	opts := prysmBasicOptions(t)
	opts = append(opts, libp2p.DialTimeout(time.Second))
	return opts
}

// Can create a new host with Service.buildOptions options
func TestPrysmOptions(t *testing.T) {
	_, err := libp2p.New(prysmBasicOptions(t)...)
	require.NoError(t, err)

	// _, err = libp2p.New(withQuic(t)...)
	// require.NoError(t, err)

	_, err = libp2p.New(withNatPort(t)...)
	require.NoError(t, err)

	_, err = libp2p.New(withResourceManager(t)...)
	require.NoError(t, err)

	_, err = libp2p.New(withAddrsFactory(t)...)
	require.NoError(t, err)
}

func TestDialTimeoutOption(t *testing.T) {
	_, err := libp2p.New(withDialTimeout(t)...)
	require.NoError(t, err)
}

func TestIdentityOptionError(t *testing.T) {
	privk1, _, err := crypto.GenerateSecp256k1Key()
	require.NoError(t, err)
	privk2, _, err := crypto.GenerateSecp256k1Key()
	require.NoError(t, err)

	opts := []libp2p.Option{
		libp2p.Identity(privk1),
		libp2p.Identity(privk2),
	}

	_, err = libp2p.New(opts...)
	require.ErrorContains(t, "cannot specify multiple identities", err)

	_, err = libp2p.New([]libp2p.Option{libp2p.Identity(&crypto.PrivErrorKey{})}...)
	require.ErrorContains(t, "test error", err)
}

func TestAddrsFactoryError(t *testing.T) {
	opts := withAddrsFactory(t)
	opts = append(opts, libp2p.AddrsFactory(func(addrs []ma.Multiaddr) []ma.Multiaddr {
		return addrs
	}))
	_, err := libp2p.New(opts...)
	require.ErrorContains(t, "cannot specify multiple address", err)
}
