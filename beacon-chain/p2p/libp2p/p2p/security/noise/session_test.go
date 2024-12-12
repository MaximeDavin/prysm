/*
Directly taken from libp2p package.
This need to be re-written
TODO(noise)
*/
package noise

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextCancellationRespected(t *testing.T) {
	initTransport := newTestTransport(t)
	respTransport := newTestTransport(t)

	init, resp := newConnPair(t)
	defer init.Close()
	defer resp.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := initTransport.SecureOutbound(ctx, init, respTransport.localID)
	require.Error(t, err)
	require.Equal(t, ctx.Err(), err)
}
