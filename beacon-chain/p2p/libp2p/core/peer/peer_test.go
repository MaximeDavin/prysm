package peer

import (
	"crypto/ecdsa"
	"testing"

	"libp2p/core/crypto"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/p2p/enode"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

const (
	testID = "16Uiu2HAkum7hhuMpWqFj3yNLcmQBGmThmqw2ohaCRThXQuKU9ohs"
	tcp    = "/ip4/192.168.0.1/tcp/5678"
	p2p    = "/p2p/" + testID
)

func TestSplitAddr(t *testing.T) {
	tpt, id := SplitAddr(nil)
	require.IsNil(t, tpt)
	require.Equal(t, "", id.String())

	tpt, id = SplitAddr(ma.StringCast(tcp + p2p))
	require.DeepEqual(t, ma.StringCast(tcp), tpt)
	require.Equal(t, testID, id.String())

	tpt, id = SplitAddr(ma.StringCast(p2p))
	require.IsNil(t, tpt)
	require.Equal(t, testID, id.String())

	tpt, id = SplitAddr(ma.StringCast(tcp))
	require.NotNil(t, tpt)
	require.Equal(t, "", id.String())
}

func TestAddrInfosFromP2pAddrs(t *testing.T) {
	infos, err := AddrInfosFromP2pAddrs()
	require.NoError(t, err)
	require.Equal(t, 0, len(infos))

	_, err = AddrInfosFromP2pAddrs(nil)
	require.ErrorContains(t, "invalid p2p multiaddr", err)

	addrs := []ma.Multiaddr{
		ma.StringCast("/ip4/128.199.219.111/tcp/4001/p2p/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64"),
		ma.StringCast("/ip4/104.236.76.40/tcp/4001/p2p/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64"),

		ma.StringCast("/p2p/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd"),
		ma.StringCast("/ip4/178.62.158.247/tcp/4001/p2p/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd"),

		ma.StringCast("/p2p/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM"),
	}
	expected := map[string][]ma.Multiaddr{
		"QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64": {
			ma.StringCast("/ip4/128.199.219.111/tcp/4001"),
			ma.StringCast("/ip4/104.236.76.40/tcp/4001"),
		},
		"QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd": {
			ma.StringCast("/ip4/178.62.158.247/tcp/4001"),
		},
		"QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM": nil,
	}
	infos, err = AddrInfosFromP2pAddrs(addrs...)
	require.NoError(t, err)
	for _, info := range infos {
		exaddrs, ok := expected[info.ID.String()]
		require.Equal(t, true, ok)
		require.Equal(t, len(exaddrs), len(info.Addrs))
	}
}

func TestAddrInfoFromString(t *testing.T) {
	_, err := AddrInfoFromString("invalid/maddr")
	require.ErrorContains(t, "failed to parse multiaddr", err)

	expected, err := AddrInfoFromP2pAddr(ma.StringCast(tcp + p2p))
	require.NoError(t, err)

	addr, err := AddrInfoFromString(tcp + p2p)
	require.NoError(t, err)
	require.DeepEqual(t, expected, addr)
}

func TestAddrInfoFromP2pAddr(t *testing.T) {
	ai, err := AddrInfoFromP2pAddr(ma.StringCast(tcp + p2p))
	require.NoError(t, err)

	require.Equal(t, 1, len(ai.Addrs))
	require.DeepEqual(t, ma.StringCast(tcp), ai.Addrs[0])
	require.Equal(t, testID, ai.ID.String())

	ai, err = AddrInfoFromP2pAddr(ma.StringCast(p2p))
	require.NoError(t, err)
	require.Equal(t, 0, len(ai.Addrs))
	require.Equal(t, testID, ai.ID.String())

	_, err = AddrInfoFromP2pAddr(ma.StringCast(tcp))
	require.ErrorIs(t, ErrInvalidAddr, err)
}

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

			m, err := Decode(tc.peerID)
			require.NoError(t, err)
			require.Equal(t, ID(m), id)
		})
	}
}

func TestIDFromPublicKeyError(t *testing.T) {

	_, err := IDFromPublicKey(&crypto.PubErrorKey{})
	require.ErrorContains(t, "test error", err)
}

func TestDecodeError(t *testing.T) {
	_, err := Decode("error")
	require.ErrorContains(t, "failed to parse peer ID", err)
}
