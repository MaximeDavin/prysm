package tests

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"runtime"
	"sync"
	"testing"
	"time"

	"libp2p"
	"libp2p/core/crypto"
	"libp2p/core/host"
	"libp2p/core/peer"
	"libp2p/core/transport"
	"libp2p/p2p/security/noise"
	"libp2p/p2p/transport/tcp"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

type TransportTestCase struct {
	Name          string
	HostGenerator func(t *testing.T) host.Host
}

// TODO(quic): add quick testcase
// TODO(noise): add noise
var transportsToTest = []TransportTestCase{
	{
		Name: "TCP / Noise(TODO) / Yamux",
		HostGenerator: func(t *testing.T) host.Host {
			privk, _, err := crypto.GenerateSecp256k1Key()
			require.NoError(t, err)

			addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
			require.NoError(t, err)

			libp2pOpts := []libp2p.Option{
				libp2p.Identity(privk),
				libp2p.ListenAddrs(addr),
				libp2p.UserAgent(version.BuildData()),
				libp2p.ConnectionGater(nil),
				libp2p.Transport(tcp.NewTCPTransport),
				libp2p.DefaultMuxers,
				libp2p.Security(noise.ID, noise.New),
				libp2p.Ping(false),
				libp2p.DisableRelay(),
			}
			h, err := libp2p.New(libp2pOpts...)
			require.NoError(t, err)
			return *h
		},
	},
}

func TestBigPing(t *testing.T) {
	// 64k buffers
	sendBuf := make([]byte, 64<<10)
	recvBuf := make([]byte, 64<<10)
	const totalSends = 64

	// Fill with random bytes
	_, err := rand.Read(sendBuf)
	require.NoError(t, err)

	for _, tc := range transportsToTest {
		t.Run(tc.Name, func(t *testing.T) {
			h1 := tc.HostGenerator(t)
			h2 := tc.HostGenerator(t)
			defer h1.Close()
			defer h2.Close()

			require.NoError(t, h2.Connect(context.Background(), peer.AddrInfo{
				ID:    h1.ID(),
				Addrs: h1.Addrs(),
			}))

			h1.SetStreamHandler("/big-ping", func(s transport.Stream) {
				io.Copy(s, s)
				s.Close()
			})

			errCh := make(chan error, 1)
			allocs := testing.AllocsPerRun(10, func() {
				s, err := h2.NewStream(context.Background(), h1.ID(), "/big-ping")
				require.NoError(t, err)
				defer s.Close()

				go func() {
					for i := 0; i < totalSends; i++ {
						_, err := io.ReadFull(s, recvBuf)
						if err != nil {
							errCh <- err
							return
						}
						if !bytes.Equal(sendBuf, recvBuf) {
							errCh <- fmt.Errorf("received data does not match sent data")
						}

					}
					_, err = s.Read([]byte{0})
					errCh <- err
				}()

				for i := 0; i < totalSends; i++ {
					s.Write(sendBuf)
				}
				// s.CloseWrite()
				s.Close()
				require.ErrorContains(t, "EOF", <-errCh)
				// require.ErrorIs(t, <-errCh, io.EOF)
			})

			if int(allocs) > (len(sendBuf)*totalSends)/4 {
				t.Logf("Expected fewer allocs, got: %f", allocs)
			}
		})
	}
}

// TestLotsOfDataManyStreams tests sending a lot of data on multiple streams.
func TestLotsOfDataManyStreams(t *testing.T) {
	// Skip on windows because of https://github.com/libp2p/go-libp2p/issues/2341
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on windows because of https://github.com/libp2p/go-libp2p/issues/2341")
	}

	// 64k buffer
	const bufSize = 64 << 10
	sendBuf := [bufSize]byte{}
	const totalStreams = 500
	const parallel = 8
	// Total sends are > 20MiB
	require.Equal(t, true, len(sendBuf)*totalStreams > 20<<20)
	// require.Greater(t, len(sendBuf)*totalStreams, 20<<20)
	t.Log("Total sends:", len(sendBuf)*totalStreams)

	// Fill with random bytes
	_, err := rand.Read(sendBuf[:])
	require.NoError(t, err)

	for _, tc := range transportsToTest {
		t.Run(tc.Name, func(t *testing.T) {
			h1 := tc.HostGenerator(t)
			h2 := tc.HostGenerator(t)
			defer h1.Close()
			defer h2.Close()
			start := time.Now()
			defer func() {
				t.Log("Total time:", time.Since(start))
			}()

			require.NoError(t, h2.Connect(context.Background(), peer.AddrInfo{
				ID:    h1.ID(),
				Addrs: h1.Addrs(),
			}))

			h1.SetStreamHandler("/big-ping", func(s transport.Stream) {
				io.Copy(s, s)
				s.Close()
			})

			sem := make(chan struct{}, parallel)
			var wg sync.WaitGroup
			for i := 0; i < totalStreams; i++ {
				wg.Add(1)
				sem <- struct{}{}
				go func() {
					defer wg.Done()
					recvBuf := [bufSize]byte{}
					defer func() { <-sem }()

					s, err := h2.NewStream(context.Background(), h1.ID(), "/big-ping")
					require.NoError(t, err)
					defer s.Close()

					_, err = s.Write(sendBuf[:])
					require.NoError(t, err)
					s.CloseWrite()

					_, err = io.ReadFull(s, recvBuf[:])
					require.NoError(t, err)
					require.Equal(t, sendBuf, recvBuf)

					_, err = s.Read([]byte{0})
					require.ErrorIs(t, err, io.EOF)
				}()
			}

			wg.Wait()
		})
	}
}

func TestManyStreams(t *testing.T) {
	const streamCount = 128
	for _, tc := range transportsToTest {
		t.Run(tc.Name, func(t *testing.T) {
			h1 := tc.HostGenerator(t)
			h2 := tc.HostGenerator(t)
			defer h1.Close()
			defer h2.Close()

			require.NoError(t, h2.Connect(context.Background(), peer.AddrInfo{
				ID:    h1.ID(),
				Addrs: h1.Addrs(),
			}))

			h1.SetStreamHandler("echo", func(s transport.Stream) {
				io.Copy(s, s)
				s.Close()
			})

			streams := make([]transport.Stream, streamCount)
			for i := 0; i < streamCount; i++ {
				s, err := h2.NewStream(context.Background(), h1.ID(), "echo")
				require.NoError(t, err)
				streams[i] = s
			}

			wg := sync.WaitGroup{}
			wg.Add(streamCount)
			errCh := make(chan error, 1)
			for _, s := range streams {
				go func(s transport.Stream) {
					defer wg.Done()

					bufw := []byte("hello")
					_, err := s.Write(bufw)
					require.NoError(t, err)

					// s.CloseWrite()
					bufr := make([]byte, len(bufw))
					_, err = s.Read(bufr)
					require.NoError(t, err)

					if err == nil {
						if !bytes.Equal(bufr, bufw) {
							err = fmt.Errorf("received data does not match sent data")
						}
					}
					if err != nil {
						select {
						case errCh <- err:
						default:
						}
					}
				}(s)
			}
			wg.Wait()
			close(errCh)

			require.NoError(t, <-errCh)
			for _, s := range streams {
				require.NoError(t, s.Close())
			}
		})
	}
}
