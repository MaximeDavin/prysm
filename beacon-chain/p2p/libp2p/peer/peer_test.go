package peer

import (
	"crypto/ecdsa"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/p2p/enode"
	mh "github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/libp2p/crypto"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

// TODO: use ConvertToInterfacePubkey located in crypto/ecdsa/utils.go when
// legacy libp2p has been replaced by this implementation
func ConvertToInterfacePubkey(pubkey *ecdsa.PublicKey) (crypto.PubKey, error) {
	xVal, yVal := new(btcec.FieldVal), new(btcec.FieldVal)
	if xVal.SetByteSlice(pubkey.X.Bytes()) {
		return nil, errors.Errorf("X value overflows")
	}
	if yVal.SetByteSlice(pubkey.Y.Bytes()) {
		return nil, errors.Errorf("Y value overflows")
	}
	newKey := crypto.PubKey((*crypto.Secp256k1PublicKey)(btcec.NewPublicKey(xVal, yVal)))
	// Zero out temporary values.
	xVal.Zero()
	yVal.Zero()
	return newKey, nil
}

func TestIDFromPublicKey(t *testing.T) {
	type testcase struct {
		name     string
		enodeUrl string
		peerID   string
	}

	testcases := []testcase{
		{
			name:     "Test Vector 1",
			enodeUrl: "enr:-MS4QA0FMDaj_hAVQgYGzpT698vUUu0sGRMgpWjziNEne44ONahIZTttUIdFS_aF3Z3qOt1baWh54Q6vCXfativyNggBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpBPtyXhYAAAOADh9QUAAAAAgmlkgnY0gmlwhKwQDAuEcXVpY4IjKYlzZWNwMjU2azGhA2CR9Vc5wWxvkpaBkNIFVxdl1apg4Pb70_hSIvslT7YliHN5bmNuZXRzAIN0Y3CCIyiDdWRwgiMo",
			peerID:   "16Uiu2HAmK9xaMgEbk6xMyWzdiu348n9as6jpwDSZsx3BYtk4c6je",
		},
		{
			name:     "Test Vector 2",
			enodeUrl: "enr:-Ku4QJsxkOibTc9FXfBWYmcdMAGwH4bnOOFb4BlTHfMdx_f0WN-u4IUqZcQVP9iuEyoxipFs7-Qd_rH_0HfyOQitc7IBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhLAJM9iJc2VjcDI1NmsxoQL2RyM26TKZzqnUsyycHQB4jnyg6Wi79rwLXtaZXty06YN1ZHCCW8w",
			peerID:   "16Uiu2HAmC13Brucnz5qR8caKi8qKK6766PFoxsF5MzK2RvbTyBRr",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			node := enode.MustParse(tc.enodeUrl)
			pubkey := node.Pubkey()

			assertedKey, err := ConvertToInterfacePubkey(pubkey)
			require.NoError(t, err)
			id, err := IDFromPublicKey(assertedKey)
			require.NoError(t, err)

			m, err := mh.FromB58String(tc.peerID)
			require.NoError(t, err)

			require.Equal(t, ID(m), id)
		})
	}
}
