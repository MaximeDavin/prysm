package libp2p

import (
	"time"

	"github.com/pkg/errors"

	"libp2p/config"
	"libp2p/core/crypto"
	"libp2p/core/host"
	"libp2p/core/peer"

	ma "github.com/multiformats/go-multiaddr"
)

type Option = config.Option

func New(opts ...Option) (*host.Host, error) {
	cfg := config.NewConfig()
	err := cfg.Apply(opts...)
	if err != nil {
		return nil, err
	}
	h, err := host.NewHost(cfg)
	return h, err
}

func DoesNothing(cfg *config.Config) error {
	return nil
}

// Identity configures libp2p to use the given private key to identify itself.
func Identity(sk crypto.PrivKey) Option {
	return func(cfg *config.Config) error {
		if cfg.PeerKey != nil {
			return errors.Errorf("cannot specify multiple identities")
		}
		peerID, err := peer.IDFromPrivateKey(sk)
		if err != nil {
			return err
		}
		cfg.PeerId = peerID
		cfg.PeerKey = sk

		return nil
	}
}

// ListenAddrs configures libp2p to listen on the given addresses.
func ListenAddrs(addrs ...ma.Multiaddr) Option {
	return func(cfg *config.Config) error {
		cfg.ListenAddrs = append(cfg.ListenAddrs, addrs...)
		return nil
	}
}

// UserAgent sets the libp2p user-agent sent along with the identify protocol
func UserAgent(userAgent string) Option {
	return func(cfg *config.Config) error {
		cfg.UserAgent = userAgent
		return nil
	}
}

// TODO(ConnectionGater)
func ConnectionGater(cg any) Option {
	return DoesNothing
}

// Transport configures libp2p to optionally use quic, TCP is always used
// TODO(quic): Activate quic in settings
func Transport(constructor any) Option {
	return DoesNothing
	// return func(cfg *config.Config) error {
	// 	typ := reflect.ValueOf(constructor).Type()
	// 	if typ == reflect.TypeOf(quic.NewQuicTransport) {
	// 		cfg.UseQuic = true
	// 	}
	// 	return nil
	// }
}

// Does nothing, it is a libp2p legacy option. Yamux is always used
func Muxer(name string, constructor any) Option {
	return DoesNothing
}

// Does nothing, it is a libp2p legacy option
var DefaultMuxers = Muxer("", nil)

// Does nothing, it is a libp2p legacy option. Noise security is always used
func Security(name string, constructor any) Option {
	return DoesNothing
}

// Does nothing, it is a libp2p legacy option.
func Ping(any) Option {
	return DoesNothing
}

// Does nothing, it is a libp2p legacy option.
func NATPortMap() Option {
	return DoesNothing
}

// TODO(relay)
func DisableRelay() Option {
	return DoesNothing
}

func AddrsFactory(factory func(addrs []ma.Multiaddr) []ma.Multiaddr) Option {
	return func(cfg *config.Config) error {
		if cfg.AddrsFactory != nil {
			return errors.Errorf("cannot specify multiple address factories")
		}
		cfg.AddrsFactory = factory
		return nil
	}
}

// ResourceManager configures libp2p to use the given ResourceManager.
// When using the p2p/host/resource-manager implementation of the ResourceManager interface,
// it is recommended to set limits for libp2p protocol by calling SetDefaultServiceLimits.
func ResourceManager(rcmgr any) Option {
	return DoesNothing
}

func DialTimeout(t time.Duration) Option {
	return func(cfg *config.Config) error {
		cfg.DialTimeout = t
		return nil
	}
}
