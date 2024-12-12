package tls

import (
	"crypto/tls"

	ic "github.com/libp2p/go-libp2p/core/crypto"
)

// Identity is used to secure connections
type Identity struct {
	config tls.Config
}

// NewIdentity creates a new identity
func NewIdentity(privKey ic.PrivKey) (*Identity, error) {
	return &Identity{
		config: tls.Config{
			MinVersion:         tls.VersionTLS13,
			InsecureSkipVerify: true, // This is not insecure here. We will verify the cert chain ourselves.
		},
	}, nil
}
