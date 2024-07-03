package generate

import (
	"fmt"
	kspace "github.com/libp2p/go-libp2p-kbucket/keyspace"
	mh "github.com/multiformats/go-multihash"
	"math/big"
)

func IsValidAccordingToIntervalRules(interval Interval, peerDistance *big.Int) bool {
	var good = false

	if peerDistance.Cmp(interval.farthest) == -1 && peerDistance.Cmp(interval.closest) > 0 {
		good = true
	}

	return good
}

func ByIntervalConfiguration(pidGenerateConfig PidGenerateConfig, targetCidKey kspace.Key) Interval {
	firstBytes, _ := mh.FromB58String(*pidGenerateConfig.FirstPeer)
	firstKey := kspace.XORKeySpace.Key(firstBytes)
	firstDistance := firstKey.Distance(targetCidKey)

	secondBytes, _ := mh.FromB58String(*pidGenerateConfig.SecondPeer)
	secondKey := kspace.XORKeySpace.Key(secondBytes)
	secondDistance := secondKey.Distance(targetCidKey)

	var interval Interval
	var left, right string
	if firstDistance.Cmp(secondDistance) > 0 {
		interval.farthest = firstDistance
		interval.closest = secondDistance

		left = *pidGenerateConfig.FirstPeer
		right = *pidGenerateConfig.SecondPeer
	} else {
		interval.farthest = secondDistance
		interval.closest = firstDistance

		left = *pidGenerateConfig.SecondPeer
		right = *pidGenerateConfig.FirstPeer
	}

	fmt.Printf("Generating %d sybils in between the peers %s and %s...\n", pidGenerateConfig.Quantity, left, right)

	return interval
}
