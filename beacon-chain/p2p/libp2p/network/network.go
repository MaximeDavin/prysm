package network

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/peer"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/peerstore"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/protocol"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/quic"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/tcp"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/transport"

	ma "github.com/multiformats/go-multiaddr"
	mss "github.com/multiformats/go-multistream"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "libp2p")

type Network interface {
	Connect(ctx context.Context, pinfo peer.AddrInfo) error
	Listen(addrs ...ma.Multiaddr) error
	NewStream(ctx context.Context, pid peer.ID, protos ...protocol.ID) (transport.Stream, error)
	DialPeer(ctx context.Context, pid peer.ID) (transport.Conn, error)
	SetStreamHandler(proto protocol.ID, handler transport.StreamHandler)
	Close()
	Peerstore() *peerstore.PeerStore
	// ConnsToPeer returns the connections in this Network for given peer.
	ConnsToPeer(p peer.ID) []transport.Conn
}

type network struct {
	conns        map[peer.ID]transport.Conn
	tcpTransport transport.Transport
	tcpListener  transport.Listener
	peerstore    *peerstore.PeerStore
	mux          *mss.MultistreamMuxer[protocol.ID]
	Stream       transport.Stream
	// quicListener transport.Listener
}

var _ Network = (*network)(nil)

func NewNetwork() *network {
	n := &network{
		conns:        make(map[peer.ID]transport.Conn),
		tcpTransport: tcp.NewTCPTransport(),
		peerstore:    peerstore.NewPeerStore(),
		mux:          mss.NewMultistreamMuxer[protocol.ID](),
	}

	return n
}

func (n *network) Connect(ctx context.Context, pinfo peer.AddrInfo) error {
	// Add peer's addresses to the store
	n.peerstore.SetAddrs(pinfo)

	// Get an existing connection else get a new one by dialing peer
	_, err := n.getConnOrDial(ctx, pinfo.ID)
	if err != nil {
		return err
	}
	return nil
}

// Check if we already have a valid connection available, if not dial and
// store the connection
func (n *network) getConnOrDial(ctx context.Context, pid peer.ID) (transport.Conn, error) {
	// Check if there is a connection already opened for this peer
	if conn, ok := n.getExistingConn(pid); ok {
		return conn, nil
	}

	// There is no available connection to the peer, let's try to get a new one
	conn, err := n.DialPeer(ctx, pid)
	if err != nil {
		return nil, err
	}

	// Store the new connection to the peer so we can re-use it later
	n.conns[pid] = conn

	return conn, nil
}

// Check if we already have a valid connection available and return it
func (n *network) getExistingConn(pid peer.ID) (transport.Conn, bool) {
	// TODO(quic): chose quic connection over tcp if possible
	c, ok := n.conns[pid]
	if ok && !c.IsClosed() {
		return c, true
	}
	return nil, false
}

func (n *network) DialPeer(ctx context.Context, pid peer.ID) (transport.Conn, error) {
	// TODO(max): put dial timeout in config
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	addrs := n.peerstore.Addrs(pid)
	if len(addrs) == 0 {
		return nil, errors.Errorf("peer %s has no addresses associated", pid)
	}
	// TODO(quic): chose quic connection over tcp if possible
	// have a function to sort in place addrs
	//
	// TODO: add retry/backoff ? not sure if needed, there is already a backoff
	// on the discovery service
	// TODO: we could dial concurrently and return the fastest
	for _, addr := range addrs {
		id, err := getTransportID(addr)
		if err != nil {
			log.WithError(err).Infof("Failed to dial peer %s with address %s", pid, addr)
			continue
		}
		if id == tcp.ID {
			if conn, err := n.tcpTransport.Dial(ctx, addr, pid); err != nil {
				log.WithError(err).Infof("Failed to dial peer %s with address %s", pid, addr)
			} else {
				return conn, nil
			}
		} else {
			return nil, errors.New("quic not supported")
		}
	}
	return nil, errors.Errorf("Failed to dial peer %s", pid)
}

func (n *network) Listen(addrs ...ma.Multiaddr) error {
	for _, addr := range addrs {
		id, err := getTransportID(addr)
		if err != nil {
			return err
		}
		if id == tcp.ID {
			l, err := n.tcpTransport.Listen(addr)
			if err != nil {
				return err
			}
			n.tcpListener = l
			go n.serve(l)
		} else {
			return errors.New("quic not supported")
		}
	}
	return nil
}

// Return the Transport matching the address
// Example: /ip4/1.2.3.4./tcp/5678 => TCP
// Example: /ip4/1.2.3.4/udp/5678/quic-v1 => QUIC
func getTransportID(addr ma.Multiaddr) (string, error) {
	hasProto := func(code int) bool {
		_, err := addr.ValueForProtocol(code)
		return err == nil
	}
	if !hasProto(ma.P_IP4) && !hasProto(ma.P_IP6) {
		return "", errors.Errorf("provided multiaddress %s is neither IP4 nor IP6", addr.String())
	}
	if !hasProto(ma.P_TCP) && !hasProto(ma.P_UDP) {
		return "", errors.Errorf("provided multiaddress %s is neither TCP nor UDP", addr.String())
	}
	if hasProto(ma.P_TCP) {
		return tcp.ID, nil
	}
	if hasProto(ma.P_UDP) {
		if hasProto(ma.P_QUIC_V1) {
			return quic.ID, nil
		} else {
			return "", errors.Errorf("provided UDP multiaddress %s does not support quic-v1", addr.String())
		}
	}
	return "", errors.Errorf("provided multiaddress %s is invalid, it must include network and transport /ip4/1.2.3.4./tcp/5678 or /ip4/1.2.3.4/udp/5678/quic-v1", addr)
}

func (n *network) serve(l transport.Listener) error {
	defer func() {
		l.Close()
	}()
	for {
		conn, err := l.Accept()
		if err != nil {
			// Log ?
			return err
		}
		// Add peer id to peerstore
		pid := conn.RemotePeer()
		n.conns[pid] = conn
		go func(conn transport.Conn) {
			defer conn.Close()
			for {
				stream, err := conn.AcceptStream()
				if err != nil {
					return
				}
				go n.mux.Handle(stream)
				n.Stream = stream
			}
		}(conn)
	}
}

// NewStream opens a new stream to given peer p, and writes a p2p/protocol
// header with given protocol.ID. If there is no connection to p, attempts
// to create one. If ProtocolID is "", writes no header.
func (n *network) NewStream(ctx context.Context, pid peer.ID, protos ...protocol.ID) (transport.Stream, error) {
	conn, err := n.getConnOrDial(ctx, pid)
	if err != nil {
		return nil, err
	}

	// TODO(max): Maybe try to check if the stream is usable. If not we can close the connection,
	// remove it from conns and attempt to make a new connection
	stream, err := conn.OpenStream(ctx)
	if err != nil {
		return nil, err
	}

	// TODO(max): Handle protocol selection

	_, err = mss.SelectOneOf(protos, stream)
	if err != nil {
		return nil, err
	}
	// stream.SetProtocol(proto)

	return stream, nil
}

func (n *network) SetStreamHandler(proto protocol.ID, handler transport.StreamHandler) {
	n.mux.AddHandler(proto, func(p protocol.ID, rwc io.ReadWriteCloser) error {
		is := rwc.(transport.Stream)
		handler(is)
		return nil
	})
}

func (n *network) Close() {
	// for _, conn := range n.conns {
	// 	conn.Close()
	// }
	// n.tcpListener.Close()
}

func (n *network) Peerstore() *peerstore.PeerStore {
	return n.peerstore
}

func (n *network) ConnsToPeer(pid peer.ID) []transport.Conn {
	conn, ok := n.conns[pid]
	if !ok {
		return []transport.Conn{}
	}
	return []transport.Conn{conn}
}
