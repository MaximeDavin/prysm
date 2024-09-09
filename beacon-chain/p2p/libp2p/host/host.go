package host

import (
	"context"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/config"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/network"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/peer"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/peerstore"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/protocol"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/transport"

	ma "github.com/multiformats/go-multiaddr"
	mss "github.com/multiformats/go-multistream"
)

type Host interface {
	// ID returns the (local) peer.ID associated with this Host
	ID() peer.ID
	// Peerstore returns the Host's repository of Peer Addresses and Keys.
	Peerstore() *peerstore.PeerStore

	// Returns the listen addresses of the Host
	Addrs() []ma.Multiaddr

	// Networks returns the Network interface of the Host
	Network() network.Network

	// Mux returns the Mux multiplexing incoming streams to protocol handlers
	Mux() *mss.MultistreamMuxer[protocol.ID]

	// Connect ensures there is a connection between this host and the peer with
	// given peer.ID. Connect will absorb the addresses in pi into its internal
	// peerstore. If there is not an active connection, Connect will issue a
	// h.Network.Dial, and block until a connection is open, or an error is
	// returned. // TODO: Relay + NAT.
	Connect(ctx context.Context, pi peer.AddrInfo) error

	// SetStreamHandler sets the protocol handler on the Host's Mux.
	// This is equivalent to:
	//   host.Mux().SetHandler(proto, handler)
	// (Thread-safe)
	SetStreamHandler(pid protocol.ID, handler transport.StreamHandler)

	// RemoveStreamHandler removes a handler on the mux that was set by
	// SetStreamHandler
	RemoveStreamHandler(pid protocol.ID)

	// NewStream opens a new stream to given peer p, and writes a p2p/protocol
	// header with given ProtocolID. If there is no connection to p, attempts
	// to create one. If ProtocolID is "", writes no header.
	// (Thread-safe)
	NewStream(ctx context.Context, pid peer.ID, protos ...protocol.ID) (transport.Stream, error)

	// Close shuts down the host, its Network, and services.
	Close() error
}

type host struct {
	network network.Network
}

var _ Host = (*host)(nil)

func NewHost(cfg *config.Config) (*host, error) {
	n, err := network.NewNetwork(cfg)
	if err != nil {
		return nil, err
	}
	h := &host{
		network: n,
	}
	return h, nil
}

func (h *host) ID() peer.ID {
	return h.network.LocalPeer()
}

func (h *host) Peerstore() *peerstore.PeerStore {
	return h.network.Peerstore()
}

func (h *host) Addrs() []ma.Multiaddr {
	return h.network.Addrs()
}

func (h *host) Network() network.Network {
	return h.network
}

func (h *host) Mux() *mss.MultistreamMuxer[protocol.ID] {
	return h.network.Mux()
}

func (h *host) Connect(ctx context.Context, pinfo peer.AddrInfo) error {
	return h.network.Connect(ctx, pinfo)
}

func (h *host) SetStreamHandler(pid protocol.ID, handler transport.StreamHandler) {
	h.network.SetStreamHandler(pid, handler)
}

func (h *host) RemoveStreamHandler(pid protocol.ID) {
	h.network.RemoveStreamHandler(pid)
}

func (h *host) NewStream(ctx context.Context, pid peer.ID, protos ...protocol.ID) (transport.Stream, error) {
	return h.network.NewStream(ctx, pid, protos...)
}

func (h *host) Close() error {
	h.network.Close()
	return nil
}
