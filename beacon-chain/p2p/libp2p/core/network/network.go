package network

import (
	"context"
	"io"
	"sync"
	"time"

	"libp2p/config"
	"libp2p/core/peer"
	"libp2p/core/peerstore"
	"libp2p/core/protocol"
	"libp2p/core/transport"
	"libp2p/p2p/transport/tcp"

	"github.com/pkg/errors"

	ma "github.com/multiformats/go-multiaddr"
	mss "github.com/multiformats/go-multistream"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "libp2p")

type conns struct {
	sync.RWMutex
	m map[peer.ID]*Conn
}

type Network struct {
	local         peer.ID
	inConns       conns
	outConns      conns
	tcpTransport  transport.Transport
	tcpListener   transport.Listener
	quicTransport transport.Transport
	quicListener  transport.Listener
	peerstore     *peerstore.PeerStore
	mux           *mss.MultistreamMuxer[protocol.ID]
	notif         Notifiee
	dialTimeout   time.Duration
}

func NewNetwork(cfg *config.Config) (*Network, error) {
	opts := tcp.DefaultTcpTransportOptions

	n := &Network{
		local:        cfg.PeerId,
		tcpTransport: tcp.NewTCPTransportWithOptions(cfg, opts),
		peerstore:    peerstore.NewPeerStore(),
		mux:          mss.NewMultistreamMuxer[protocol.ID](),
		notif:        &NoopNotifiee{},
		dialTimeout:  cfg.DialTimeout,
	}

	n.inConns.m = make(map[peer.ID]*Conn)
	n.outConns.m = make(map[peer.ID]*Conn)

	// TODO(quic)
	// if cfg.UseQuic {
	// 	n.quicTransport = quic.NewQuicTransport()
	// }

	err := n.Listen(cfg.ListenAddrs...)
	if err != nil {
		return nil, err
	}

	return n, nil
}

// Connect ensures there is a connection between this host and the peer with
// given peer.ID. If there is not an active connection, Connect will issue a
// h.Network.Dial, and block until a connection is open, or an error is returned.
// Connect will absorb the addresses in pi into its internal peerstore.
func (n *Network) Connect(ctx context.Context, pinfo peer.AddrInfo) error {
	// Add peer's addresses to the store
	n.peerstore.SetAddrs(pinfo)

	// Check if there is an existing outbound connection else get a new one by dialing peer
	if _, ok := n.getExistingConn(pinfo.ID, &n.outConns); ok {
		return nil
	}

	// There is no available outbound connection to the peer, let's try to get a new one
	_, err := n.DialPeer(ctx, pinfo.ID)
	if err != nil {
		return err
	}

	return nil
}

// Check if we already have a valid connection available, if not dial and
// store the connection
// func (n *Network) getConnOrDial(ctx context.Context, pid peer.ID) (*Conn, error) {
// 	// Check if there is an outbound connection already opened for this peer
// 	if conn, ok := n.getExistingConn(pid); ok {
// 		return conn, nil
// 	}

// 	// There is no available outbound connection to the peer, let's try to get a new one
// 	conn, err := n.DialPeer(ctx, pid)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return conn, nil
// }

// Store the connection to this peer
func (n *Network) addConn(pid peer.ID, conn *Conn, dir transport.Direction) {
	var conns *conns
	if dir == transport.DirInbound {
		conns = &n.inConns
	} else {
		conns = &n.outConns
	}

	// Store the new connection to the peer so we can re-use it later
	conns.Lock()
	conns.m[pid] = conn
	conns.Unlock()

	// Call notifier connected callback
	n.notif.Connected(*n, *conn)

}

func getConn(conns *conns, pid peer.ID) (*Conn, bool) {
	conns.RLock()
	defer conns.RUnlock()

	conn, ok := conns.m[pid]
	return conn, ok
}

// Check if we already have a valid connection available and return it
func (n *Network) getExistingConn(pid peer.ID, conns ...*conns) (*Conn, bool) {
	for _, c := range conns {
		conn, ok := getConn(c, pid)
		if ok && !conn.conn.IsClosed() {
			return conn, true
		}
	}
	return nil, false
}

func (n *Network) DialPeer(ctx context.Context, pid peer.ID) (*Conn, error) {
	addrs := n.peerstore.Addrs(pid)
	if len(addrs) == 0 {
		return nil, errors.Errorf("peer %s has no addresses associated", pid)
	}
	// TODO(quic): chose quic connection over tcp if possible
	// have a function to sort in place addrs
	//
	// TODO: add retry/backoff ? not sure if needed, there is already a backoff
	// on the discovery service
	for _, addr := range addrs {
		ctx, cancel := context.WithTimeout(ctx, n.dialTimeout)
		defer cancel()

		id, err := GetTransportID(addr)
		if err != nil {
			log.WithError(err).Infof("Failed to dial peer %s with address %s, transport protocol not supported", pid, addr)
			continue
		}
		if id == tcp.ID {
			if conn, err := n.tcpTransport.Dial(ctx, addr, pid); err != nil {
				log.WithError(err).Infof("Failed to dial peer %s with address %s", pid, addr)
			} else {
				c := NewConn(conn, n, transport.DirOutbound)
				n.addConn(pid, c, transport.DirOutbound)
				return c, nil
			}
		} else {
			return nil, errors.New("quic not supported")
		}
	}
	return nil, errors.Errorf("Failed to dial peer %s", pid)
}

func (n *Network) Listen(addrs ...ma.Multiaddr) error {
	for _, addr := range addrs {
		id, err := GetTransportID(addr)
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
			// TODO(quic): handle quic listener
			return errors.New("quic not supported")
		}
	}
	return nil
}

func (n *Network) ListenAddresses() []ma.Multiaddr {
	return []ma.Multiaddr{n.tcpListener.Multiaddr()}
}

// Return the Transport matching the address
// Example: /ip4/1.2.3.4./tcp/5678 => TCP
// Example: /ip4/1.2.3.4/udp/5678/quic-v1 => QUIC
func GetTransportID(addr ma.Multiaddr) (string, error) {
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
	// TODO(quic)
	// if hasProto(ma.P_UDP) {
	// 	if hasProto(ma.P_QUIC_V1) {
	// 		return quic.ID, nil
	// 	} else {
	// 		return "", errors.Errorf("provided UDP multiaddress %s does not support quic-v1", addr.String())
	// 	}
	// }
	return "", errors.Errorf("provided multiaddress %s is invalid, it must include network and transport /ip4/1.2.3.4./tcp/5678", addr)
}

func (n *Network) serve(l transport.Listener) error {
	defer func() {
		l.Close()
	}()
	for {
		conn, err := l.Accept()
		if err != nil {
			// Log ?
			return err
		}
		// Store the connection
		c := NewConn(conn, n, transport.DirInbound)
		n.addConn(conn.RemotePeer(), c, transport.DirInbound)

		go func(conn transport.UpgradedConn) {
			defer conn.Close()
			for {
				stream, err := conn.AcceptStream()
				if err != nil {
					return
				}
				go n.mux.Handle(stream)
			}
		}(conn)
	}
}

// NewStream opens a new stream to given peer p, and writes a p2p/protocol
// header with given protocol.ID. If there is no connection to p, attempts
// to create one. If ProtocolID is "", writes no header.
func (n *Network) NewStream(ctx context.Context, pid peer.ID, protos ...protocol.ID) (*Stream, error) {
	// Check if there is an existing connection else get a new one by dialing peer
	c, ok := n.getExistingConn(pid, n.getConns()...)
	if !ok {
		conn, err := n.DialPeer(ctx, pid)
		if err != nil {
			return nil, err
		}
		c = conn
	}

	// TODO(max): Maybe try to check if the stream is usable. If not we can close the connection,
	// remove it from conns and attempt to make a new connection
	stream, err := c.conn.OpenStream(ctx)
	if err != nil {
		return nil, err
	}

	// TODO(max): Handle protocol selection
	proto, err := mss.SelectOneOf(protos, stream)
	if err != nil {
		return nil, err
	}

	return NewStream(stream, c, proto, transport.DirOutbound), nil
}

func (n *Network) SetStreamHandler(proto protocol.ID, handler StreamHandler) {
	n.mux.AddHandler(proto, func(p protocol.ID, rwc io.ReadWriteCloser) error {
		is := rwc.(transport.Stream)
		handler(is)
		return nil
	})
}

func (n *Network) RemoveStreamHandler(proto protocol.ID) {
	n.mux.RemoveHandler(proto)
}

// Close closes all connection to each peer and all listeners.
func (n *Network) Close() error {
	// We close all connections
	conns := n.Conns()
	for _, c := range conns {
		c.conn.Close()
	}

	// We close all listeners
	n.tcpListener.Close()
	if n.quicTransport != nil {
		n.quicListener.Close()
	}

	return nil
}

// ClosePeer closes all connections to the given peer.
func (n *Network) ClosePeer(pid peer.ID) error {
	connsm := n.getConns()
	n.lockConns()
	defer n.unlockConns()

	for _, conns := range connsm {
		if conn, ok := conns.m[pid]; ok {
			go func() {
				conn.Close()
				delete(conns.m, pid)
			}()
		}
	}
	return nil
}

// TODO: Use this version with error handling if we find a way to test it
// (ie find a way to make conn.Close() return an error)
// func (n *Network) ClosePeer(pid peer.ID) error {
// 	connsm := n.getConns()
// 	n.lockConns()
// 	defer n.unlockConns()

// 	errCh := make(chan error)
// 	for _, conns := range connsm {
// 		if conn, ok := conns.m[pid]; ok {
// 			go func() {
// 				errCh <- conn.Close()
// 				delete(conns.m, pid)
// 			}()
// 		}
// 	}

// 	var errs []string
// 	for range connsm {
// 		err := <-errCh
// 		if err != nil {
// 			errs = append(errs, err.Error())
// 		}
// 	}

// 	if len(errs) > 0 {
// 		return fmt.Errorf("when disconnecting from peer %s: %s", pid, strings.Join(errs, ", "))
// 	}
// 	return nil
// }

func (n *Network) Peerstore() *peerstore.PeerStore {
	return n.peerstore
}

func tLen(conns []*conns) int {
	res := 0
	for _, c := range conns {
		res += len(c.m)
	}
	return res
}

func (n *Network) Peers() []peer.ID {
	connsm := n.getConns()
	n.rLockConns()
	defer n.rUnlockConns()

	peers := make([]peer.ID, 0, tLen(connsm))

	for _, conns := range connsm {
		for p := range conns.m {
			peers = append(peers, p)
		}
	}
	return peers
}

// func (n *Network) Peers() []peer.ID {
// 	n.conns.RLock()
// 	defer n.conns.RUnlock()

// 	peers := make([]peer.ID, 0, len(n.conns.m))
// 	for p := range n.conns.m {
// 		peers = append(peers, p)
// 	}
// 	return peers
// }

func (n *Network) getConns() []*conns {
	return []*conns{&n.inConns, &n.outConns}
}

func (n *Network) rLockConns() {
	for _, c := range n.getConns() {
		c.RLock()
	}
}

func (n *Network) rUnlockConns() {
	for _, c := range n.getConns() {
		c.RUnlock()
	}
}

func (n *Network) lockConns() {
	for _, c := range n.getConns() {
		c.Lock()
	}
}

func (n *Network) unlockConns() {
	for _, c := range n.getConns() {
		c.Unlock()
	}
}

func (n *Network) ConnsToPeer(pid peer.ID) []*Conn {
	connsm := n.getConns()
	n.rLockConns()
	defer n.rUnlockConns()

	res := make([]*Conn, 0, len(connsm))
	for _, conns := range connsm {
		if conn, ok := conns.m[pid]; ok {
			res = append(res, conn)
		}
	}
	return res
}

// conns := connsm[pid]
// res := make([]*Conn, len(conns))
// copy(res, conns)

// return res
// }

func (n *Network) Conns() []*Conn {
	// n.conns.RLock()
	// defer n.conns.RUnlock()

	// var res = make([]*Conn, 0)

	// for _, conns := range n.conns.m {
	// 	res = append(res, conns...)
	// }
	// return res
	connsm := n.getConns()
	n.rLockConns()
	defer n.rUnlockConns()

	res := make([]*Conn, 0, len(connsm))
	for _, conns := range connsm {
		for _, conn := range conns.m {
			res = append(res, conn)
		}
	}
	return res
}

func (n *Network) LocalPeer() peer.ID {
	return n.local
}

func (n *Network) Addrs() []ma.Multiaddr {
	// TODO(quic): Return quicListener addr too
	return []ma.Multiaddr{n.tcpListener.Multiaddr()}
}

func (n *Network) Mux() *mss.MultistreamMuxer[protocol.ID] {
	return n.mux
}

// ErrReset is returned when reading or writing on a reset stream.
var ErrReset = errors.New("stream reset")

// Notify signs up Notifiee to receive signals when events happen
func (n *Network) Notify(f Notifiee) {
	n.notif = f
}
