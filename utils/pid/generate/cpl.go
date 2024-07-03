package generate

import (
	"fmt"
	"github.com/ipfs/go-cid"
	kb "github.com/libp2p/go-libp2p-kbucket"
	kspace "github.com/libp2p/go-libp2p-kbucket/keyspace"
	"github.com/libp2p/go-libp2p/core/peer"
	mh "github.com/multiformats/go-multihash"
	"math/big"
)

func IsValidAccordingToCPLRules(cpl int, interval Interval, key KeyInfo, targetCidKey kspace.Key) bool {
	good := false

	peerCPL := kb.CommonPrefixLen(targetCidKey.Bytes, key.peerKey.Bytes)

	if peerCPL != cpl {
		return false
	}

	if interval.closest == nil || key.peerDistance.Cmp(interval.closest) == -1 {
		good = true
	}

	return good
}

func getClosestInCPL(cid cid.Cid, peers []string, cpl int) string {
	targetBytes := []byte(kb.ConvertKey(string(cid.Hash())))
	peerBytes := make([][]byte, len(peers))
	for i := range peerBytes {
		decode, _ := peer.Decode(peers[i])
		peerBytes[i] = kb.ConvertKey(string(decode))
	}

	var cplPeers []string

	for i, peerByte := range peerBytes {
		prefixLen := kb.CommonPrefixLen(targetBytes, peerByte)

		if prefixLen == cpl {
			cplPeers = append(cplPeers, peers[i])
		}
	}

	if len(cplPeers) == 0 {
		fmt.Printf("No peers in the CPL %d, generating any nodes in this CPL.\n\n", cpl)

		return ""
	}

	fmt.Printf("Peers in CPL %d (%d): %s\n", cpl, len(cplPeers), cplPeers)
	closest, distance := GetClosestToCIDFromPeersList(cid, cplPeers)
	fmt.Printf("Closest in CPL %d: %s (distance: %d)\n\n", cpl, closest, distance)

	return closest
}

func ByCplConfiguration(pidGenerateConfig PidGenerateConfig, cid cid.Cid, targetCidKey kspace.Key, closestList []string) Interval {
	var closestInCplDistance *big.Int

	closestInCpl := getClosestInCPL(cid, closestList, pidGenerateConfig.Cpl)

	if closestInCpl != "" {
		closestByte, _ := mh.FromB58String(closestInCpl)
		closestKey := kspace.XORKeySpace.Key(closestByte)
		closestInCplDistance = closestKey.Distance(targetCidKey)
	}

	if closestInCpl == "" {
		fmt.Printf("Generating %d sybils in the CPL %d...\n", pidGenerateConfig.Quantity, pidGenerateConfig.Cpl)
	} else {
		fmt.Printf("Generating %d sybil identities closer than %s in the CPL %d...\n", pidGenerateConfig.Quantity,
			closestInCpl, pidGenerateConfig.Cpl)
	}

	return Interval{closest: closestInCplDistance}
}
