package generate

import (
	"fmt"
	kspace "github.com/libp2p/go-libp2p-kbucket/keyspace"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-base32"
	mh "github.com/multiformats/go-multihash"
	"strings"
)

func IsValidAccordingToBase32Rules(referencePeer string, key KeyInfo) bool {
	good := false

	referencePeerId, _ := peer.Decode(referencePeer)
	referencePeerEncoded := base32.RawStdEncoding.EncodeToString([]byte(referencePeerId))

	keyPeerId, _ := peer.Decode(key.peerId)
	keyPeerEncoded := base32.RawStdEncoding.EncodeToString([]byte(keyPeerId))

	if strings.Compare(keyPeerEncoded, referencePeerEncoded) < 0 {
		good = true
    // fmt.Println(keyPeerEncoded, ">", referencePeerEncoded)
	}

	return good
}

func IsValidAccordingToClosestRules(interval Interval, key KeyInfo) bool {
	good := false

	if key.peerDistance.Cmp(interval.closest) == -1 {
		good = true
	}

	return good
}

func ByClosestConfiguration(pidGenerateConfig PidGenerateConfig, targetCidKey kspace.Key, closestList []string) Interval {
	closest := closestList[0]

	closestByte, _ := mh.FromB58String(closest)
	closestKey := kspace.XORKeySpace.Key(closestByte)
	closestDistance := closestKey.Distance(targetCidKey)

	fmt.Printf("Generating %d sybils closer than the cpl peer (%s)...\n", pidGenerateConfig.Quantity, closest)

	return Interval{farthest: nil, closest: closestDistance}
}
