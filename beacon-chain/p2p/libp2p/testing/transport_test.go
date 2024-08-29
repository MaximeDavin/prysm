package testing_test

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/peer"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/tcp"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/transport"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func TestTcpTransport(t *testing.T) {
	var Subtests = []func(t *testing.T, ta, tb transport.Transport, maddr ma.Multiaddr, peerA peer.ID){
		SubtestBasic,
		SubtestCancel,
		SubtestPingPong,
		// Stolen from the stream muxer test suite.
		SubtestStress1Conn1Stream1Msg,
		SubtestStress1Conn1Stream100Msg,
		SubtestStress1Conn100Stream100Msg,
		SubtestStressManyConn10Stream50Msg,
		SubtestStress1Conn1000Stream10Msg,
		SubtestStress1Conn100Stream100Msg10MB,
		SubtestStreamOpenStress,
		// SubtestStreamReset,
	}

	ta, err := tcp.NewTCPTransport()
	require.NoError(t, err)
	tb, err := tcp.NewTCPTransport()
	require.NoError(t, err)

	zero := "/ip4/127.0.0.1/tcp/0"
	maddr, err := ma.NewMultiaddr(zero)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range Subtests {
		t.Run(getFunctionName(f), func(t *testing.T) {
			f(t, ta, tb, maddr, "peerA")
		})
	}
}
