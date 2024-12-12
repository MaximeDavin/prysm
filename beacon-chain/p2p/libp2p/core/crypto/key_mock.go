package crypto

import (
	"github.com/libp2p/go-libp2p/core/crypto/pb"

	"github.com/pkg/errors"
)

type PubErrorKey struct{}

var _ PubKey = &PubErrorKey{}

func (k *PubErrorKey) Equals(Key) bool {
	return true
}

func (k *PubErrorKey) Raw() ([]byte, error) {
	return []byte{}, errors.Errorf("test error")
}

func (k *PubErrorKey) Type() pb.KeyType {
	return 3
}

func (k *PubErrorKey) Verify(data []byte, sigStr []byte) (success bool, err error) {
	return true, nil
}

type PrivErrorKey struct{}

var _ PrivKey = &PrivErrorKey{}

func (k *PrivErrorKey) Raw() ([]byte, error) {
	return []byte{}, errors.Errorf("test error")
}

func (k *PrivErrorKey) Type() pb.KeyType {
	return 3
}

func (k *PrivErrorKey) GetPublic() PubKey {
	return &PubErrorKey{}
}

func (k *PrivErrorKey) Equals(Key) bool {
	return true
}

func (k *PrivErrorKey) Sign(data []byte) ([]byte, error) {
	return data, nil
}
