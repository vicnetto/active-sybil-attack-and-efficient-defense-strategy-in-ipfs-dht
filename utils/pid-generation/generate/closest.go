package generate

import (
	"fmt"
	kspace "github.com/libp2p/go-libp2p-kbucket/keyspace"
	mh "github.com/multiformats/go-multihash"
)

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
