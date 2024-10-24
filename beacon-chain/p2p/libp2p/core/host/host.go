package host

import (
	"context"

	"libp2p/config"
	"libp2p/core/network"
	"libp2p/core/peer"
	"libp2p/core/peerstore"
	"libp2p/core/protocol"
	"libp2p/core/transport"

	ma "github.com/multiformats/go-multiaddr"
	mss "github.com/multiformats/go-multistream"
)

type Host struct {
	network *network.Network
}

func NewHost(cfg *config.Config) (*Host, error) {
	n, err := network.NewNetwork(cfg)
	if err != nil {
		return nil, err
	}
	h := &Host{
		network: n,
	}
	return h, nil
}

// ID returns the (local) peer.ID associated with this Host
func (h *Host) ID() peer.ID {
	return h.network.LocalPeer()
}

// Peerstore returns the Host's repository of Peer Addresses and Keys.
func (h *Host) Peerstore() *peerstore.PeerStore {
	return h.network.Peerstore()
}

// Returns the listen addresses of the Host
func (h *Host) Addrs() []ma.Multiaddr {
	return h.network.Addrs()
}

// Networks returns the Network interface of the Host
func (h *Host) Network() *network.Network {
	return h.network
}

// Mux returns the Mux multiplexing incoming streams to protocol handlers
func (h *Host) Mux() *mss.MultistreamMuxer[protocol.ID] {
	return h.network.Mux()
}

// Connect ensures there is a connection between this Host and the peer with
// given peer.ID. Connect will absorb the addresses into its internal
// peerstore.
// If there is not an active connection, Connect will issue a
// h.Network.Dial, and block until a connection is open, or an error is
// returned.
// TODO: Relay + NAT.
func (h *Host) Connect(ctx context.Context, pinfo peer.AddrInfo) error {
	return h.network.Connect(ctx, pinfo)
}

// SetStreamHandler sets the protocol handler on the Host's Mux.
// This is equivalent to:
//
//	Host.Mux().SetHandler(proto, handler)
//
// (Thread-safe)
func (h *Host) SetStreamHandler(pid protocol.ID, handler network.StreamHandler) {
	h.network.SetStreamHandler(pid, handler)
}

// RemoveStreamHandler removes a handler on the mux that was set by
// SetStreamHandler
func (h *Host) RemoveStreamHandler(pid protocol.ID) {
	h.network.RemoveStreamHandler(pid)
}

// NewStream opens a new stream to given peer p, and writes a p2p/protocol
// header with given ProtocolID. If there is no connection to p, attempts
// to create one. If ProtocolID is "", writes no header.
// (Thread-safe)
func (h *Host) NewStream(ctx context.Context, pid peer.ID, protos ...protocol.ID) (transport.Stream, error) {
	return h.network.NewStream(ctx, pid, protos...)
}

// Close shuts down the Host, its Network, and services.
func (h *Host) Close() error {
	return h.network.Close()
}
