package tcp

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/muxer/yamux"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/network"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/peer"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/security/noise"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/transport"

	manet "github.com/multiformats/go-multiaddr/net"
	mss "github.com/multiformats/go-multistream"
)

type conn struct {
	transport.MuxedConn
	transport.SecureConnMixin
	transport.ConnMultiaddrsMixin
}

var _ transport.UpgradedConn = &conn{}

func Upgrade(ctx context.Context, t *TcpTransport, rawConn manet.Conn, pid peer.ID, direction network.Direction) (transport.UpgradedConn, error) {
	sconn, err := upgradeSecurity(ctx, t, rawConn, pid, direction)
	if err != nil {
		return nil, err
	}

	mconn, err := upgradeStream(t, sconn, pid, direction)
	if err != nil {
		return nil, err
	}

	return &conn{
		MuxedConn:           mconn,
		SecureConnMixin:     sconn,
		ConnMultiaddrsMixin: rawConn,
	}, nil
}

func upgradeSecurity(ctx context.Context, t *TcpTransport, rawConn manet.Conn, pid peer.ID, direction network.Direction) (transport.SecureConn, error) {
	log.Debugf("peer %s security protocol negociation in progress...", pid)

	var proto string
	var err error
	switch direction {
	case network.DirInbound:
		proto, _, err = t.SecurityMuxer.Negotiate(rawConn)
	case network.DirOutbound:
		proto = t.SecuritySupported
		err = mss.SelectProtoOrFail(t.SecuritySupported, rawConn)
	default:
		panic("programming error: invalid direction")
	}

	if err != nil {
		return nil, err
	}
	if proto != noise.ID {
		return nil, errors.Errorf("security protocol not supported, only %s is supported", noise.ID)
	}
	sconn, err := noise.Secure(ctx, rawConn, direction, pid)
	if err != nil {
		return nil, err
	}
	log.Debugf("peer %s security protocol negociation succeded", pid)

	return sconn, nil
}

func upgradeStream(t *TcpTransport, sconn transport.SecureConn, pid peer.ID, direction network.Direction) (transport.MuxedConn, error) {
	log.Debugf("peer %s stream multiplexing protocol negociation in progress...", pid)

	var proto string
	var err error
	switch direction {
	case network.DirInbound:
		proto, _, err = t.StreamMuxer.Negotiate(sconn)
	case network.DirOutbound:
		proto = t.StreamSupported
		err = mss.SelectProtoOrFail(t.StreamSupported, sconn)
	default:
		panic("programming error: invalid direction")
	}

	if err != nil {
		return nil, err
	}
	if proto != yamux.ID {
		return nil, errors.Errorf("stream multiplexing protocol not supported, only %s is supported", yamux.ID)
	}
	upconn, err := yamux.Multiplex(sconn, direction)
	if err != nil {
		return nil, err
	}
	log.Debugf("peer %s stream multiplexing protocol negociation succeded", pid)

	return upconn, nil
}
