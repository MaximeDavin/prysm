package tests

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"testing"

	"libp2p/core/peer"
	"libp2p/core/transport"
	"libp2p/p2p/transport/tcp"

	ma "github.com/multiformats/go-multiaddr"
)

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func TestTcpTransport(t *testing.T) {
	var Subtests = []func(t *testing.T, ta, tb transport.Transport, maddr ma.Multiaddr, peerA peer.ID){
		SubtestBasic,
		SubtestCancel,
		SubtestPingPong,
		// Stolen from the stream muxer test suite.
		SubtestStress1Conn1Stream1Msg,
		SubtestStress1Conn1Stream100Msg,
		SubtestStress1Conn100Stream100Msg,
		SubtestStressManyConn10Stream50Msg,
		SubtestStress1Conn1000Stream10Msg,
		SubtestStress1Conn100Stream100Msg10MB,
		SubtestStreamOpenStress,
		// SubtestStreamReset,
	}

	cfg := CreateConfig(t)
	ta := tcp.NewTCPTransport(cfg)
	tb := tcp.NewTCPTransport(cfg)

	zero := "/ip4/127.0.0.1/tcp/0"
	maddr, err := ma.NewMultiaddr(zero)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range Subtests {
		t.Run(getFunctionName(f), func(t *testing.T) {
			f(t, ta, tb, maddr, cfg.PeerId)
		})
	}
}

var testData = []byte("this is some test data")

func SubtestBasic(t *testing.T, ta, tb transport.Transport, maddr ma.Multiaddr, peerA peer.ID) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	list, err := ta.Listen(maddr)
	if err != nil {
		t.Fatal(err)
	}
	defer list.Close()

	var (
		connA, connB transport.UpgradedConn
		done         = make(chan struct{})
	)
	defer func() {
		<-done
		if connA != nil {
			connA.Close()
		}
		if connB != nil {
			connB.Close()
		}
	}()

	go func() {
		defer close(done)
		var err error
		connB, err = list.Accept()

		if err != nil {
			t.Error(err)
			return
		}
		s, err := connB.AcceptStream()
		if err != nil {
			t.Error(err)
			return
		}

		buf := make([]byte, len(testData))
		_, err = s.Read(buf)
		if err != nil {
			t.Error(err)
			return
		}

		if !bytes.Equal(testData, buf) {
			t.Errorf("expected %s, got %s", testData, buf)
		}

		n, err := s.Write(testData)
		if err != nil {
			t.Error(err)
			return
		}
		if n != len(testData) {
			t.Error(err)
			return
		}

		err = s.Close()
		if err != nil {
			t.Error(err)
		}
	}()

	connA, err = tb.Dial(ctx, list.Multiaddr(), peerA)
	if err != nil {
		t.Fatal(err)
	}

	s, err := connA.OpenStream(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	n, err := s.Write(testData)
	if err != nil {
		t.Fatal(err)
		return
	}

	if n != len(testData) {
		t.Fatalf("failed to write enough data (a->b)")
		return
	}

	// TODO: Uncomment when CloseWrite is ready and change Read with io.ReadAll
	// if err = s.CloseWrite(); err != nil {
	// 	t.Fatal(err)
	// 	return
	// }

	buf := make([]byte, len(testData))
	_, err = s.Read(buf)
	if err != nil {
		t.Fatal(err)
		return
	}

	if !bytes.Equal(testData, buf) {
		t.Errorf("expected %s, got %s", testData, buf)
	}

	if err = s.Close(); err != nil {
		t.Fatal(err)
		return
	}
}

func SubtestPingPong(t *testing.T, ta, tb transport.Transport, maddr ma.Multiaddr, peerA peer.ID) {
	streams := 100

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	list, err := ta.Listen(maddr)
	if err != nil {
		t.Fatal(err)
	}
	defer list.Close()

	var (
		connA, connB transport.UpgradedConn
	)
	defer func() {
		if connA != nil {
			connA.Close()
		}
		if connB != nil {
			connB.Close()
		}
	}()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		connA, err = list.Accept()
		if err != nil {
			t.Error(err)
			return
		}

		var sWg sync.WaitGroup
		for i := 0; i < streams; i++ {
			s, err := connA.AcceptStream()
			if err != nil {
				t.Error(err)
				return
			}

			sWg.Add(1)
			go func() {
				defer sWg.Done()

				buf := make([]byte, len(testData)+5)
				_, err := s.Read(buf)
				if err != nil {
					s.Close()
					t.Error(err)
					return
				}
				if !bytes.HasPrefix(buf, testData) {
					t.Errorf("expected %q to have prefix %q", string(buf), string(testData))
				}

				n, err := s.Write(buf)
				if err != nil {
					s.Close()
					t.Error(err)
					return
				}

				if n != len(buf) {
					s.Close()
					t.Error(err)
					return
				}
				s.Close()
			}()
		}
		sWg.Wait()
	}()

	connB, err = tb.Dial(ctx, list.Multiaddr(), peerA)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < streams; i++ {
		s, err := connB.OpenStream(context.Background())
		if err != nil {
			t.Error(err)
			continue
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			data := []byte(fmt.Sprintf("%s - %2d", testData, i))
			n, err := s.Write(data)
			if err != nil {
				s.Close()
				t.Error(err)
				return
			}

			if n != len(data) {
				s.Close()
				t.Error("failed to write enough data (a->b)")
				return
			}

			// TODO: Uncomment when CloseWrite is ready and change Read with io.ReadAll
			// if err = s.CloseWrite(); err != nil {
			// 	t.Error(err)
			// 	return
			// }

			buf := make([]byte, len(data))
			_, err = s.Read(buf)
			if err != nil {
				s.Close()
				t.Error(err)
				return
			}
			if !bytes.Equal(data, buf) {
				t.Errorf("expected %q, got %q", string(data), string(buf))
			}

			if err = s.Close(); err != nil {
				t.Error(err)
				return
			}
		}(i)
	}
	wg.Wait()
}

func SubtestCancel(t *testing.T, ta, tb transport.Transport, maddr ma.Multiaddr, peerA peer.ID) {
	list, err := ta.Listen(maddr)
	if err != nil {
		t.Fatal(err)
	}
	defer list.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c, err := tb.Dial(ctx, list.Multiaddr(), peerA)
	if err == nil {
		c.Close()
		t.Fatal("dial should have failed")
	}
}
