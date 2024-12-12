package tcp

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/config"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/transport"
	"github.com/libp2p/go-libp2p/direction"
	"github.com/libp2p/go-libp2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/security/noise"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	mss "github.com/multiformats/go-multistream"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithFields(logrus.Fields{
	"prefix":    "p2p",
	"transport": "TCP",
})

const ID = "TCP"

type TcpTransport struct {
	SecurityMuxer *mss.MultistreamMuxer[string]
	StreamMuxer   *mss.MultistreamMuxer[string]
	Noise         *noise.Transport
	TcpTransportOptions
}

type TcpTransportOptions struct {
	// Maximum duration between accepting the raw TCP connection and returning
	// a fully upgraded connection
	AcceptTimeout time.Duration
	// ID of security protocol proposed in the protocol selection
	SecuritySupported string
	// ID of stream multiplexing protocol proposed in the protocol selection
	StreamsSupported string
	// Function that upgrade a raw TCP connection into a fully upgraded connection
	// We will always use the one provided by default, except for some test cases
	// Upgrade func(ctx context.Context, t *TcpTransport, rawConn manet.Conn, pid peer.ID, direction network.Direction) (transport.UpgradedConn, error)
}

var DefaultTcpTransportOptions TcpTransportOptions = TcpTransportOptions{
	AcceptTimeout:     15 * time.Second,
	SecuritySupported: noise.ID,
	StreamsSupported:  yamux.ID,
}

var _ transport.Transport = &TcpTransport{}

func NewTCPTransport(cfg *config.Config) *TcpTransport {
	return NewTCPTransportWithOptions(cfg, DefaultTcpTransportOptions)
}

func NewTCPTransportWithOptions(cfg *config.Config, options TcpTransportOptions) *TcpTransport {
	t := &TcpTransport{
		SecurityMuxer:       mss.NewMultistreamMuxer[string](),
		StreamMuxer:         mss.NewMultistreamMuxer[string](),
		TcpTransportOptions: options,
	}
	t.SecurityMuxer.AddHandler(options.SecuritySupported, nil)
	t.StreamMuxer.AddHandler(options.StreamsSupported, nil)

	n, _ := noise.New(ID, cfg.PeerKey, options.StreamsSupported)
	t.Noise = n
	return t
}

// Dial the given multiaddr and upgrade the resulting outbound connection
func (t *TcpTransport) Dial(ctx context.Context, addr ma.Multiaddr, pid peer.ID) (transport.UpgradedConn, error) {
	var d manet.Dialer
	conn, err := d.DialContext(ctx, addr)
	if err != nil {
		return nil, err
	}
	return Upgrade(ctx, t, conn, pid, direction.DirOutbound)
}
