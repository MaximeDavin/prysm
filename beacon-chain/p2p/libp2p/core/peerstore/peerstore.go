package peerstore

import (
	"github.com/libp2p/go-libp2p/core/peer"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "github.com/libp2p/go-libp2p")

type PeerStore struct {
	addrs map[peer.ID][]ma.Multiaddr
}

func NewPeerStore() *PeerStore {
	ps := &PeerStore{}
	ps.addrs = make(map[peer.ID][]ma.Multiaddr)
	return ps
}

// Store addresses of a peer into the peerstore. It replaces previously
// stored addresses.
// Basic verification and uniqueness check is done on each address
func (ps *PeerStore) SetAddrs(pinfo peer.AddrInfo) {
	ps.addrs[pinfo.ID] = make([]ma.Multiaddr, 0)

	// ma.Unique remove duplicates
	for _, addr := range ma.Unique(pinfo.Addrs) {
		// Remove suffix of /p2p/peer-id from address
		addr, addrPid := peer.SplitAddr(addr)
		if addr == nil {
			log.Warnf("was passed nil multiaddr for peer: %s", pinfo.ID)
			continue
		}
		if addrPid != "" && addrPid != pinfo.ID {
			log.Warnf("was passed p2p address with a different peerId. found: %s, expected: %s", addrPid, pinfo.ID)
			continue
		}
		ps.addrs[pinfo.ID] = append(ps.addrs[pinfo.ID], addr)
	}
}

func (ps *PeerStore) Addrs(pid peer.ID) []ma.Multiaddr {
	addrs, ok := ps.addrs[pid]
	if !ok {
		return []ma.Multiaddr{}
	}
	return addrs
}

// DEPECRATED
func (ps *PeerStore) PeerInfo(pid peer.ID) peer.AddrInfo {
	return peer.AddrInfo{ID: pid}
}
