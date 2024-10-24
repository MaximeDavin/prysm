package crypto

import (
	"testing"

	"libp2p/core/crypto/pb"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestScep256k1Key(t *testing.T) {
	priv, pub, err := GenerateSecp256k1Key()
	require.NoError(t, err)
	require.DeepEqual(t, pub, priv.GetPublic())

	_, err = priv.Raw()
	require.NoError(t, err)
	_, err = pub.Raw()
	require.NoError(t, err)

	keyType := priv.Type()
	require.Equal(t, pb.KeyType_Secp256k1, keyType)
	keyType = pub.Type()
	require.Equal(t, pb.KeyType_Secp256k1, keyType)

	_, err = MarshalPublicKey(pub)
	require.NoError(t, err)
	_, err = MarshalPublicKey(&PubErrorKey{})
	require.ErrorContains(t, "test error", err)
}
