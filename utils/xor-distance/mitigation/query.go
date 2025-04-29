package mitigation

import (
	"context"
	"fmt"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/core"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/qpeerset"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	kspace "github.com/libp2p/go-libp2p-kbucket/keyspace"
	"github.com/libp2p/go-libp2p/core/peer"
	mh "github.com/multiformats/go-multihash"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"math/big"
	"time"
)

type Result struct {
	minCpl      int
	maxDistance *big.Int
}

func PerformRandomLookupReturningQueriedPeers(ctx context.Context, clientNode *core.IpfsNode) (cid.Cid, []peer.ID) {
	var allPeersReceived *qpeerset.QueryPeerset
	var cidDecode cid.Cid

	for {
		randomId, err := clientNode.DHT.WAN.RoutingTable().GenRandPeerID(0)
		if err != nil {
			log.Error.Println("Error while generating a random peer ID:", err)
		}

		cidDecode, err = cid.Decode(randomId.String())
		if err != nil {
			log.Error.Println("Error while decoding random generated PID:", err)
		}

		ctxTimeout, ctxTimeoutCancel := context.WithTimeout(ctx, 120*time.Second)
		log.Info.Printf("  Getting closest peers to %s...", cidDecode.String())
		_, allPeersReceived, err = clientNode.DHT.WAN.GetPathClosestPeers(ctxTimeout, string(cidDecode.Hash()))
		if err != nil || allPeersReceived == nil {
			log.Error.Println("Error while asking the closest peers to the CID:", err)
			log.Error.Println("Retrying...")
			ctxTimeoutCancel()
			continue
		}

		ctxTimeoutCancel()
		break
	}

	return cidDecode, allPeersReceived.GetClosestInStates(qpeerset.PeerHeard, qpeerset.PeerWaiting, qpeerset.PeerQueried)
}

func PerformRandomLookupReturningQueriedPeersWithFullInformation(ctx context.Context, clientNode *core.IpfsNode) (cid.Cid, dht.LookupWithFollowupResult) {
	var lookupResult *dht.LookupWithFollowupResult
	var cidDecode cid.Cid

	for {
		randomId, err := clientNode.DHT.WAN.RoutingTable().GenRandPeerID(0)
		if err != nil {
			log.Error.Println("Error while generating a random peer ID:", err)
		}

		cidDecode, err = cid.Decode(randomId.String())
		if err != nil {
			log.Error.Println("Error while decoding random generated PID:", err)
		}

		ctxTimeout, ctxTimeoutCancel := context.WithTimeout(ctx, 120*time.Second)
		log.Info.Printf("  Getting closest peers to %s...", cidDecode.String())
		lookupResult, err = clientNode.DHT.WAN.GetPathClosestPeersFullInformation(ctxTimeout, string(cidDecode.Hash()))
		if err != nil || lookupResult == nil || lookupResult.AllPeersContacted == nil {
			log.Error.Println("Error while asking the closest peers to the CID:", err)
			log.Error.Println("Retrying...")
			ctxTimeoutCancel()
			continue
		}

		ctxTimeoutCancel()
		break
	}

	return cidDecode, *lookupResult
}

func CalculateAverageDistancePerPeerQuantity(ctx context.Context, maxPeerQuantity int) []WelfordAverage {
	var averageMaxDistance []WelfordAverage

	for i := 0; i < maxPeerQuantity; i++ {
		averageMaxDistance = append(averageMaxDistance, WelfordAverage{})
	}

	var alreadyQueriedPeers []peer.ID
	for peerQuantity := 0; peerQuantity < maxPeerQuantity; peerQuantity++ {
		peerConfig := ipfspeer.ConfigForNormalClient(8080)
		_, clientNode, err := ipfspeer.SpawnEphemeral(ctx, peerConfig)
		if err != nil {
			log.Error.Println("Error instantiating the clientNode:", err.Error())
			panic(err)
		}

		log.Info.Println("PID is UP:", clientNode.Identity.String())

		log.Info.Println("Sleep for 10 seconds before starting...")
		time.Sleep(10 * time.Second)
		fmt.Println()

		distanceAverage, err := GetFarthestKAverage(ctx, clientNode, peerQuantity+1, &alreadyQueriedPeers, nil, "")
		if err != nil {
			log.Error.Println("Error getting farthest k average:", err.Error())
			peerQuantity--
			continue
		}

		averageMaxDistance[peerQuantity] = distanceAverage

		fmt.Println()
	}

	return averageMaxDistance
}

type QuantityPerAverage map[MeanType]int

func NewQuantityPerAverage() QuantityPerAverage {
	quantityPerAverage := QuantityPerAverage{}
	for meanType := MeanType(0); meanType <= LastMeanType; meanType++ {
		quantityPerAverage[meanType] = 0
	}
	return quantityPerAverage
}

func CountPeersPerAverage(cidDecode cid.Cid, maxDistancePerPeerQuantity []WelfordAverage,
	contactedPeers []peer.ID, peersPerDistance *map[WelfordAverage]QuantityPerAverage) {

	targetCIDByte, _ := mh.FromB58String(cidDecode.String())
	targetCIDKey := kspace.XORKeySpace.Key(targetCIDByte)

	for _, currentPeer := range contactedPeers {
		peerByte, _ := mh.FromB58String(currentPeer.String())
		peerKey := kspace.XORKeySpace.Key(peerByte)
		distance := peerKey.Distance(targetCIDKey)
		cpl := kbucket.CommonPrefixLen(targetCIDKey.Bytes, peerKey.Bytes)

		for _, maxDistance := range maxDistancePerPeerQuantity {
			currentInfo := (*peersPerDistance)[maxDistance]

			if distance.Cmp(maxDistance.GetAverage(Mean)) < 0 {
				currentInfo[Mean] = currentInfo[Mean] + 1
			}

			if distance.Cmp(maxDistance.GetAverage(MeanStdDev)) < 0 {
				currentInfo[MeanStdDev] = currentInfo[MeanStdDev] + 1
			}

			if distance.Cmp(maxDistance.GetAverage(WeightedMean)) < 0 {
				currentInfo[WeightedMean] = currentInfo[WeightedMean] + 1
			}

			if distance.Cmp(maxDistance.GetAverage(WeightedMeanStdDev)) < 0 {
				currentInfo[WeightedMeanStdDev] = currentInfo[WeightedMeanStdDev] + 1
			}

			if cpl >= int(maxDistance.GetAverage(CPL).Int64()) {
				currentInfo[CPL] = currentInfo[CPL] + 1
			}

			(*peersPerDistance)[maxDistance] = currentInfo
		}

		// for mt := MeanType(0); mt <= LastMeanType; mt++ {
		// 	currentInfo := (*peersPerDistance)

		// 	toPrint += fmt.Sprintf(" %s: %d (%s);", mt.String(), peersPerDistance[distance][mt],
		// 		mitigation.ToSciNotation(distance.GetAverage(mt)))
		// }
	}

	// Minimum should be k=20, as in a standard scenario at least 20 will be sent.
	for _, maxDistance := range maxDistancePerPeerQuantity {
		currentInfo := (*peersPerDistance)[maxDistance]

		if currentInfo[Mean] < 20 {
			currentInfo[Mean] = 20
		}

		if currentInfo[MeanStdDev] < 20 {
			currentInfo[MeanStdDev] = 20
		}

		if currentInfo[WeightedMean] < 20 {
			currentInfo[WeightedMean] = 20
		}

		if currentInfo[WeightedMeanStdDev] < 20 {
			currentInfo[WeightedMeanStdDev] = 20
		}

		if currentInfo[CPL] < 20 {
			currentInfo[CPL] = 20
		}

		(*peersPerDistance)[maxDistance] = currentInfo
	}
}
