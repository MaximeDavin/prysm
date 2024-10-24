package network_test

import (
	"testing"
	"time"

	"libp2p/core/network"
	"libp2p/core/protocol"
	"libp2p/core/transport"
	tests_utils "libp2p/p2p/testing"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"go.uber.org/mock/gomock"
)

func TestStreamInterface(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := tests_utils.NewMockStream(ctrl)
	s := network.NewStream(m, nil, "", transport.DirInbound)

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
	c := network.NewConn(nil, nil, transport.DirInbound)
	s := network.NewStream(nil, c, "", transport.DirInbound)

	require.Equal(t, c, s.Conn())
}

func TestStreamProtocol(t *testing.T) {
	s := network.NewStream(nil, nil, "init", transport.DirInbound)
	require.Equal(t, protocol.ID("init"), s.Protocol())
	s.SetProtocol("modified")
	require.Equal(t, protocol.ID("modified"), s.Protocol())
}

func TestStreamStat(t *testing.T) {
	s := network.NewStream(nil, nil, "", transport.DirInbound)

	require.Equal(t, network.Stats{Direction: transport.DirInbound}, s.Stat())
}
