package tcp

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/transport"
	"github.com/libp2p/go-libp2p/direction"
	"github.com/libp2p/go-libp2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/security/noise"

	"github.com/pkg/errors"

	manet "github.com/multiformats/go-multiaddr/net"
	mss "github.com/multiformats/go-multistream"
)

type conn struct {
	transport.MuxedConn
	transport.ConnSecure
	transport.ConnMultiaddrs
}

var _ transport.UpgradedConn = &conn{}

func Upgrade(ctx context.Context, t *TcpTransport, rawConn manet.Conn, pid peer.ID, dir direction.Direction) (transport.UpgradedConn, error) {
	if err := rawConn.SetDeadline(time.Now().Add(t.TcpTransportOptions.AcceptTimeout)); err != nil {
		return nil, err
	}

	sconn, err := upgradeSecurity(ctx, t, rawConn, pid, dir)
	if err != nil {
		return nil, err
	}

	mconn, err := upgradeStream(t, sconn, pid, dir)
	if err != nil {
		return nil, err
	}

	// stat := Stats{Direction: direction}

	c := &conn{
		MuxedConn:      mconn,
		ConnSecure:     sconn,
		ConnMultiaddrs: rawConn,
		// stat:                stat,
	}

	return c, nil
}

func negotiateProtocol(muxer *mss.MultistreamMuxer[string], supported string, conn io.ReadWriteCloser, dir direction.Direction) (string, error) {
	if dir == direction.DirInbound {
		proto, _, err := muxer.Negotiate(conn)
		if err != nil && errors.Is(err, io.EOF) {
			return "", fmt.Errorf("connection closed by dialer, possibly due to protocols not supported: %v", muxer.Protocols())
		}
		if err != nil {
			return "", err
		}
		return proto, nil
	} else {
		err := mss.SelectProtoOrFail(supported, conn)
		if err != nil {
			conn.Close()
			return "", err
		}
		return supported, nil
	}
}

func secure(ctx context.Context, t *TcpTransport, conn manet.Conn, pid peer.ID, dir direction.Direction) (transport.SecureConn, error) {
	if dir == direction.DirInbound {
		s, err := t.Noise.SecureInbound(ctx, conn, pid)
		if err != nil {
			return nil, err
		}
		return s, nil
	} else {
		s, err := t.Noise.SecureOutbound(ctx, conn, pid)
		if err != nil {
			return nil, err
		}
		return s, nil
	}
}

func upgradeSecurity(ctx context.Context, t *TcpTransport, rawConn manet.Conn, pid peer.ID, dir direction.Direction) (transport.SecureConn, error) {
	log.Debugf("peer %s security protocol negociation in progress...", pid)

	proto, err := negotiateProtocol(t.SecurityMuxer, t.SecuritySupported, rawConn, dir)
	if err != nil {
		return nil, err
	}

	if proto != noise.ID {
		return nil, errors.Errorf("security protocol not supported, only %s is supported", noise.ID)
	}
	sconn, err := secure(ctx, t, rawConn, pid, dir)
	if err != nil {
		return nil, err
	}
	log.Debugf("peer %s security protocol negociation succeded", pid)

	return sconn, nil
}

func upgradeStream(t *TcpTransport, sconn transport.SecureConn, pid peer.ID, dir direction.Direction) (transport.MuxedConn, error) {
	log.Debugf("peer %s stream multiplexing protocol negotiation in progress...", pid)

	proto, err := negotiateProtocol(t.StreamMuxer, t.StreamsSupported, sconn, dir)
	if err != nil {
		return nil, err
	}

	if proto != yamux.ID {
		return nil, errors.Errorf("stream multiplexing protocol not supported, only %s is supported", yamux.ID)
	}
	upconn, err := yamux.Multiplex(sconn, dir)
	if err != nil {
		return nil, err
	}
	log.Debugf("peer %s stream multiplexing protocol negotiation succeeded", pid)

	return upconn, nil
}
