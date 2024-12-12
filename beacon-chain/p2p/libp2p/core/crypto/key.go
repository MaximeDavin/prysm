package crypto

import (
	// "crypto/ecdsa"
	"crypto/sha256"
	"crypto/subtle"

	"google.golang.org/protobuf/proto"

	"github.com/libp2p/go-libp2p/core/crypto/pb"

	"github.com/pkg/errors"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

// Key represents a crypto key that can be compared to another key
type Key interface {
	// Equals checks whether two PubKeys are the same
	Equals(Key) bool

	// Raw returns the raw bytes of the key (not wrapped in the
	// libp2p-crypto protobuf).
	//
	// This function is the inverse of {Priv,Pub}KeyUnmarshaler.
	Raw() ([]byte, error)

	// Type returns the protobuf key type.
	Type() pb.KeyType
}

// PubKey is a public key that can be used to verify data signed with the corresponding private key
type PubKey interface {
	Key

	// Verify that 'sig' is the signed hash of 'data'
	Verify(data []byte, sig []byte) (bool, error)
}

// PrivKey represents a private key that can be used to generate a public key and sign data
type PrivKey interface {
	Key

	// Return a public key paired with this private key
	GetPublic() PubKey

	// Cryptographically sign the given bytes
	Sign([]byte) ([]byte, error)
}

// Secp256k1PrivateKey is a Secp256k1 private key
type Secp256k1PrivateKey secp256k1.PrivateKey

// Secp256k1PublicKey is a Secp256k1 public key
type Secp256k1PublicKey secp256k1.PublicKey

// GenerateSecp256k1Key generates a new Secp256k1 private and public key pair
func GenerateSecp256k1Key() (PrivKey, PubKey, error) {
	privk, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return nil, nil, err
	}

	k := (*Secp256k1PrivateKey)(privk)
	return k, k.GetPublic(), nil
}

// MarshalPublicKey converts a public key object into a protobuf serialized
// public key
func MarshalPublicKey(k PubKey) ([]byte, error) {

	data, err := k.Raw()
	if err != nil {
		return nil, err
	}
	pbmes := &pb.PublicKey{
		Type: k.Type().Enum(),
		Data: data,
	}
	return proto.Marshal(pbmes)
}

// UnmarshalPublicKey converts a protobuf serialized public key into its
// representative object
func UnmarshalPublicKey(data []byte) (PubKey, error) {
	pmes := new(pb.PublicKey)
	err := proto.Unmarshal(data, pmes)
	if err != nil {
		return nil, err
	}

	res := pmes.GetData()
	k, err := secp256k1.ParsePubKey(res)

	if err != nil {
		return nil, err
	}

	return (*Secp256k1PublicKey)(k), nil
}

// Type returns the private key type
func (k *Secp256k1PrivateKey) Type() pb.KeyType {
	return pb.KeyType_Secp256k1
}

// Equals compares two private keys
func (k *Secp256k1PrivateKey) Equals(o Key) bool {
	sk, ok := o.(*Secp256k1PrivateKey)
	if !ok {
		return basicEquals(k, o)
	}

	return k.GetPublic().Equals(sk.GetPublic())
}

// Sign returns a signature from input data
func (k *Secp256k1PrivateKey) Sign(data []byte) (_sig []byte, err error) {
	// TODO: handle panic
	// defer func() { catch.HandlePanic(recover(), &err, "secp256k1 signing") }()
	key := (*secp256k1.PrivateKey)(k)
	hash := sha256.Sum256(data)
	sig := ecdsa.Sign(key, hash[:])

	return sig.Serialize(), nil
}

// Raw returns the bytes of the key
func (k *Secp256k1PrivateKey) Raw() ([]byte, error) {
	return (*secp256k1.PrivateKey)(k).Serialize(), nil
}

// GetPublic returns a public key
func (k *Secp256k1PrivateKey) GetPublic() PubKey {
	return (*Secp256k1PublicKey)((*secp256k1.PrivateKey)(k).PubKey())
}

func basicEquals(k1, k2 Key) bool {
	if k1.Type() != k2.Type() {
		return false
	}

	a, err := k1.Raw()
	if err != nil {
		return false
	}
	b, err := k2.Raw()
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}

// Equals compares two public keys
func (k *Secp256k1PublicKey) Equals(o Key) bool {
	sk, ok := o.(*Secp256k1PublicKey)
	if !ok {
		return basicEquals(k, o)
	}

	return (*secp256k1.PublicKey)(k).IsEqual((*secp256k1.PublicKey)(sk))
}

func (k *Secp256k1PublicKey) Raw() (res []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("secp256k1 public-key marshaling")
		}
	}()
	return (*secp256k1.PublicKey)(k).SerializeCompressed(), nil
}

func (k *Secp256k1PublicKey) Type() pb.KeyType {
	return pb.KeyType_Secp256k1
}

// Verify compares a signature against the input data
func (k *Secp256k1PublicKey) Verify(data []byte, sigStr []byte) (success bool, err error) {
	defer func() {
		// To be extra safe.
		if err != nil {
			success = false
		}
	}()
	sig, err := ecdsa.ParseDERSignature(sigStr)
	if err != nil {
		return false, err
	}

	hash := sha256.Sum256(data)
	return sig.Verify(hash[:], (*secp256k1.PublicKey)(k)), nil
}
