package transport

import (
	"context"
	"io"
	"net"

	"libp2p/core/crypto"
	"libp2p/core/peer"

	ma "github.com/multiformats/go-multiaddr"
)

// A UpgradedConn represents a connection that has been secured and multiplexed
type UpgradedConn interface {
	MuxedConn
	SecureConnMixin
	ConnMultiaddrsMixin
}

// ConnSecurity is the interface that one can mix into a connection interface to
// give it the security methods.
type SecureConnMixin interface {
	// LocalPeer returns our peer ID
	LocalPeer() peer.ID
	// RemotePeer returns the peer ID of the remote peer.
	RemotePeer() peer.ID
	// RemotePublicKey returns the public key of the remote peer.
	RemotePublicKey() crypto.PubKey
}

// A UpgradedConn represents a connection that has been secured
type SecureConn interface {
	net.Conn
	SecureConnMixin
}

// ConnMultiaddrs is an interface mixin for connection types that provide multiaddr
// addresses for the endpoints.
type ConnMultiaddrsMixin interface {
	// LocalMultiaddr returns the local Multiaddr associated
	// with this connection
	LocalMultiaddr() ma.Multiaddr

	// RemoteMultiaddr returns the remote Multiaddr associated
	// with this connection
	RemoteMultiaddr() ma.Multiaddr
}

// MuxedConn represents a connection to a remote peer that has been
// extended to support stream multiplexing.
//
// A MuxedConn allows a single net.Conn connection to carry many logically
// independent bidirectional streams of binary data.
type MuxedConn interface {
	// Close closes the stream muxer and the the underlying net.Conn.
	io.Closer
	// OpenStream creates a new stream.
	OpenStream(context.Context) (Stream, error)
	// AcceptStream accepts a stream opened by the other side.
	AcceptStream() (Stream, error)
	// IsClosed returns whether a connection is fully closed, so it can
	// be garbage collected.
	IsClosed() bool
}
