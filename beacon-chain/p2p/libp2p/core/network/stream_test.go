package network_test

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/direction"
	tests_utils "github.com/libp2p/go-libp2p/p2p/testing"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"go.uber.org/mock/gomock"
)

func TestStreamInterface(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := tests_utils.NewMockStream(ctrl)
	s := network.NewStream(m, nil, "", direction.DirInbound)

	testStr := []byte("test")

	m.EXPECT().Read(testStr).Return(4, nil).Times(1)
	s.Read(testStr)

	m.EXPECT().Write(testStr).Return(4, nil).Times(1)
	s.Write(testStr)

	m.EXPECT().Close().Return(nil).Times(1)
	s.Close()

	m.EXPECT().Reset().Return(nil).Times(1)
	s.Reset()

	m.EXPECT().CloseWrite().Return(nil).Times(1)
	s.CloseWrite()

	testTime := time.Now()

	m.EXPECT().SetDeadline(testTime).Return(nil).Times(1)
	s.SetDeadline(testTime)

	m.EXPECT().SetReadDeadline(testTime).Return(nil).Times(1)
	s.SetReadDeadline(testTime)

	m.EXPECT().SetWriteDeadline(testTime).Return(nil).Times(1)
	s.SetWriteDeadline(testTime)

}

func TestStreamConn(t *testing.T) {
	c := network.NewConn(nil, nil, direction.DirInbound)
	s := network.NewStream(nil, c, "", direction.DirInbound)

	require.Equal(t, c, s.Conn())
}

func TestStreamProtocol(t *testing.T) {
	s := network.NewStream(nil, nil, "init", direction.DirInbound)
	require.Equal(t, protocol.ID("init"), s.Protocol())
	s.SetProtocol("modified")
	require.Equal(t, protocol.ID("modified"), s.Protocol())
}

func TestStreamStat(t *testing.T) {
	s := network.NewStream(nil, nil, "", direction.DirInbound)

	require.Equal(t, network.Stats{Direction: direction.DirInbound}, s.Stat())
}
