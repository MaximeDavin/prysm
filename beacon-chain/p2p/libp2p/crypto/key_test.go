package crypto

import (
	"encoding/hex"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestIDFromPublicKey(t *testing.T) {
	type testcase struct {
		name   string
		data   string
		peerID string
		error  string
	}

	testcases := []testcase{
		{
			name:   "secp256k1 Peer ID",
			data:   "d02d03e3334fdd708876713ee3b2d7655bcc7ef66557cdd5719b6b0697bf259c",
			peerID: "16Uiu2HAmK9xaMgEbk6xMyWzdiu348n9as6jpwDSZsx3BYtk4c6je",
		},
		{
			name:  "Invalid certificate",
			data:  "308201773082011da003020102020830a73c5d896a1109300a06082a8648ce3d04030230003020170d3735303130313030303030305a180f34303936303130313030303030305a30003059301306072a8648ce3d020106082a8648ce3d03010703420004bbe62df9a7c1c46b7f1f21d556deec5382a36df146fb29c7f1240e60d7d5328570e3b71d99602b77a65c9b3655f62837f8d66b59f1763b8c9beba3be07778043a37f307d307b060a2b0601040183a25a01010101ff046a3068042408011220ec8094573afb9728088860864f7bcea2d4fd412fef09a8e2d24d482377c20db60440ecabae8354afa2f0af4b8d2ad871e865cb5a7c0c8d3dbdbf42de577f92461a0ebb0a28703e33581af7d2a4f2270fc37aec6261fcc95f8af08f3f4806581c730a300a06082a8648ce3d040302034800304502202dfb17a6fa0f94ee0e2e6a3b9fb6e986f311dee27392058016464bd130930a61022100ba4b937a11c8d3172b81e7cd04aedb79b978c4379c2b5b24d565dd5d67d3cb3c",
			error: "signature invalid",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := hex.DecodeString(tc.data)
			require.NoError(t, err)
			UnmarshalPublicKey(data)
			t.Log(data)

			// cert, err := x509.ParseCertificate(data)
			require.NoError(t, err)
			// key, err := PubKeyFromCertChain([]*x509.Certificate{cert})
			// if tc.error != "" {
			// 	require.Error(t, err)
			// 	require.Contains(t, err.Error(), tc.error)
			// 	return
			// }
			// require.NoError(t, err)
			// id, err := peer.IDFromPublicKey(key)
			// require.NoError(t, err)
			// expectedID, err := peer.Decode(tc.peerID)
			// require.NoError(t, err)
			// require.Equal(t, expectedID, id)
		})
	}
}
