package cpl

import (
	"context"
	"fmt"
	gocid "github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/core"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/vicnetto/active-sybil-attack/logger"
	"time"

	kb "github.com/libp2p/go-libp2p-kbucket"
)

var KeySize = 256

var log = logger.InitializeLogger()

func GeneratePidAndGetClosest(ctx context.Context, node *core.IpfsNode) (gocid.Cid, []peer.ID) {
	var cid gocid.Cid
	var peers []peer.ID

	for {
		// Generate random peer using the Kubo function
		randomPid, err := node.DHT.WAN.RoutingTable().GenRandPeerID(0)
		if err != nil {
			fmt.Println(err)
			continue
		}

		cid, err = gocid.Decode(randomPid.String())
		if err != nil {
			fmt.Println(err)
			continue
		}
		log.Info.Printf("Getting closest peers to %s...\n\n", cid.String())

		timeoutCtx, cancelTimeoutCtx := context.WithTimeout(ctx, 30*time.Second)

		// Get the closest peers to verify the CPL of each one
		peers, err = node.DHT.WAN.GetClosestPeers(timeoutCtx, string(cid.Hash()))
		if err != nil {
			log.Error.Printf("  %s", err)
			cancelTimeoutCtx()
			continue
		}

		cancelTimeoutCtx()
		break
	}

	return cid, peers
}

func GeneratePidAndGetClosestAsString(ctx context.Context, node *core.IpfsNode) (gocid.Cid, []string) {
	cid, closest := GeneratePidAndGetClosest(ctx, node)

	var closestAsString []string
	for _, pid := range closest {
		closestAsString = append(closestAsString, pid.String())
	}

	return cid, closestAsString
}

func GetCurrentClosest(ctx context.Context, cid gocid.Cid, node *core.IpfsNode, timeout time.Duration) ([]peer.ID, error) {
	var peers []peer.ID
	var err error

	for {
		ctxTimeout, cancelCtxTimeout := context.WithTimeout(ctx, timeout)

		log.Info.Printf("Getting closest peers to %s...", cid.String())
		peers, err = node.DHT.WAN.GetClosestPeers(ctxTimeout, string(cid.Hash()))

		if err != nil {
			log.Error.Printf("  %s.. trying again!", err)
			cancelCtxTimeout()
			continue
		}

		cancelCtxTimeout()
		break
	}

	return peers, nil
}

func GetCurrentClosestAsString(ctx context.Context, cid gocid.Cid, node *core.IpfsNode, timeout time.Duration) ([]string, error) {
	closest, err := GetCurrentClosest(ctx, cid, node, timeout)
	if err != nil {
		return nil, err
	}

	var closestAsString []string
	for _, pid := range closest {
		closestAsString = append(closestAsString, pid.String())
	}

	return closestAsString, nil
}

func CountInCpl(cid gocid.Cid, closestPeersToCid []string) []int {
	counts := make([]int, KeySize)

	// Convert CID to hash and after to bytes
	cidHash := cid.Hash()
	targetCid := kb.ConvertKey(string(cidHash))

	// Compare the two values and count how many appearances per CPL
	for _, pid := range closestPeersToCid {
		pidDecoded, _ := peer.Decode(pid)
		pidBytes := kb.ConvertKey(string(pidDecoded))
		prefixLen := kb.CommonPrefixLen(targetCid, pidBytes)
		counts[prefixLen]++
	}

	return counts
}
