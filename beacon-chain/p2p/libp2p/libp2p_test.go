package libp2p_test

import (
	"testing"

	mplex "github.com/libp2p/go-libp2p-mplex"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/crypto"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/security/noise"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/testing/require"

	ma "github.com/multiformats/go-multiaddr"
)

// TODO(max): Add quic, natport, relay, hostadrdress, dns
func prysmOptions(t *testing.T) []libp2p.Option {
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
		libp2p.Transport(nil),
		libp2p.DefaultMuxers,
		libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Ping(false),
		libp2p.DisableRelay(),
	}
	return opts
}

// Can create a new host with Service.buildOptions options
func TestPrysmOptions(t *testing.T) {
	_, err := libp2p.New(prysmOptions(t)...)
	require.NoError(t, err)
}
