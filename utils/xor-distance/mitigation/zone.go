package mitigation

import (
	"context"
	"fmt"
	"github.com/ipfs/kubo/core"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	kspace "github.com/libp2p/go-libp2p-kbucket/keyspace"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-base32"
	mh "github.com/multiformats/go-multihash"
	"github.com/vicnetto/active-sybil-attack/logger"
	"math"
	"math/big"
	"math/rand"
	"os"
	"slices"
	"time"
)

var log = logger.InitializeLogger()

func removePeersFromList(base []peer.ID, remove []peer.ID) []peer.ID {
	for _, r := range remove {
		if index := slices.Index(base, r); index != -1 {
			base = append(base[:index], base[index+1:]...)
		}
	}

	return base
}

func QueryPeerForKClosestFromItself(ctx context.Context, clientNode *core.IpfsNode, pid peer.ID) ([]peer.ID, error) {
	// After asking directly from the peer, we need to know its addresses. As the peer is already in the RT, probably
	// we already know this information, but its probably better to be sure.
	if _, err := clientNode.DHT.WAN.FindPeer(ctx, pid); err != nil {
		return nil, err
	}

	// Create new routing table to allow generation of CIDs within an CPL.
	pidMultiHash, _ := mh.FromB58String(pid.String())
	rt, err := kbucket.NewRoutingTable(20, kbucket.ConvertKey(string(pidMultiHash)), time.Minute, clientNode.Peerstore, time.Minute, nil)
	if err != nil {
		return nil, err
	}

	randomId, err := rt.GenRandPeerID(15)
	// log.Info.Println("Random PID)", randomId)

	// Ask directly the peer for the closest nodes he knows for the random generated peer in the closest CPL as possible.
	closest, err := clientNode.DHT.WAN.ProtoMessenger.GetClosestPeers(ctx, pid, randomId)
	if err != nil {
		return nil, err
	}

	// Remove garbage addresses, returning only the peer.ID
	var closestPeerId []peer.ID
	for _, peerInfo := range closest {
		closestPeerId = append(closestPeerId, peerInfo.ID)
	}

	return closestPeerId, nil
}

// getFarthestDistance returns the CPL and distance of the farthest k node of the closest list.
func getFarthestDistance(target string, closest []peer.ID, print bool) (int, *big.Int) {
	targetCIDByte, _ := mh.FromB58String(target)
	targetCIDKey := kspace.XORKeySpace.Key(targetCIDByte)

	var minCpl int
	var maxDistance *big.Int
	if print {
		log.Info.Printf("Closest from %s: [", target)
	}
	for i, node := range closest {
		peerMultiHash, _ := mh.FromB58String(node.String())
		peerKey := kspace.XORKeySpace.Key(peerMultiHash)

		cpl := kbucket.CommonPrefixLen(targetCIDKey.Bytes, peerKey.Bytes)
		if i == 0 || cpl < minCpl {
			minCpl = cpl
		}

		distance := peerKey.Distance(targetCIDKey)
		if i == 0 || distance.Cmp(maxDistance) > 0 {
			maxDistance = distance
		}

		if print {
			log.Info.Printf(" {%d, (CPL: %d) (Distance: %s) %s (%s)}", i+1, cpl, ToSciNotation(distance), node.String(), base32.RawStdEncoding.EncodeToString([]byte(node)))
		}
	}
	if print {
		log.Info.Println("]")
		fmt.Println()
	}

	return minCpl, maxDistance
}

// ToSciNotation returns a string with the scientific notation of the big.Int.
func ToSciNotation(x *big.Int) string {
	if x.Cmp(big.NewInt(0)) == 0 {
		return "0"
	}

	if x.Cmp(big.NewInt(int64(keySpace))) < 0 {
		return x.String()
	}

	// Get the absolute value of the big.Int
	absValue := new(big.Int).Abs(x)

	// Convert to float64 for calculation of scientific notation
	floatValue := new(big.Float).SetInt(absValue)
	mantissa := new(big.Float)
	exponent := float64(0)

	// Normalize the float to [1,10) range
	for floatValue.Cmp(big.NewFloat(10)) >= 0 {
		floatValue.Quo(floatValue, big.NewFloat(10))
		exponent++
	}

	for floatValue.Cmp(big.NewFloat(1)) < 0 {
		floatValue.Mul(floatValue, big.NewFloat(10))
		exponent--
	}

	// Format mantissa and exponent as a string
	mantissa.Float64()
	sign := ""
	if x.Sign() < 0 {
		sign = "-"
	}

	return fmt.Sprintf("%s%.3fe%d", sign, floatValue, int(exponent))
}

func getValidPeerToQuery(ctx context.Context, clientNode *core.IpfsNode, alreadyQueriedPeers []peer.ID) peer.ID {
	var nextPeerToQuery peer.ID

	for {
		peersInRT := clientNode.DHT.WAN.RoutingTable().ListPeers()
		validPeersInRT := removePeersFromList(peersInRT, alreadyQueriedPeers)

		// In case there are no more valid peers to query, perform a random DHT query.
		if len(validPeersInRT) == 0 {
			log.Info.Printf("Already asked for all the nodes in the RT. Performing random DHT request...")

			ctxTimeout, cancelCtxTimeout := context.WithTimeout(ctx, 10*time.Second)
			id, err := clientNode.DHT.WAN.RoutingTable().GenRandPeerID(0)
			if err != nil {
				log.Error.Println("Error generating random peer:", err)
				panic(err)
			}

			_, err = clientNode.DHT.WAN.GetClosestPeers(ctxTimeout, id.String())
			if err != nil && !os.IsTimeout(err) {
				log.Error.Println("Error filling the RT using a GetClosestPeers:", err)
				cancelCtxTimeout()
				continue
			}

			cancelCtxTimeout()
			continue
		}

		if len(validPeersInRT) == 0 {
			continue
		}

		// In case there are valid peers available, get a random one.
		randomPosition := rand.Intn(len(validPeersInRT))
		nextPeerToQuery = validPeersInRT[randomPosition]
		if slices.Contains(alreadyQueriedPeers, nextPeerToQuery) {
			continue
		}

		break
	}

	return nextPeerToQuery
}

func GetFarthestKAverage(ctx context.Context, clientNode *core.IpfsNode, nodesToContact int, alreadyQueriedPeers *[]peer.ID) (WelfordAverage, error) {
	if alreadyQueriedPeers == nil {
		alreadyQueriedPeers = &[]peer.ID{}
	}

	var minCplResponse []int
	var maxDistanceResponse []*big.Int
	var maxDistanceResponseStd = NewWelfordMovingAverage()
	minCplAverage := float64(0)

	log.Info.Printf("Obtaining minCpl and maxDistance by contacting %d peers...", nodesToContact)

	var currentPeer peer.ID
	for peersContacted := 0; peersContacted < nodesToContact; peersContacted++ {
		currentPeer = getValidPeerToQuery(ctx, clientNode, *alreadyQueriedPeers)
		*alreadyQueriedPeers = append(*alreadyQueriedPeers, currentPeer)

		log.Info.Printf("%d) Querying %s for their closest peers...", peersContacted+1, currentPeer.String())
		ctxTimeout, cancelTimeout := context.WithTimeout(ctx, 3*time.Minute)
		queryClosest, err := QueryPeerForKClosestFromItself(ctxTimeout, clientNode, currentPeer)
		if err != nil {
			log.Error.Println("Error while querying the peer:", err.Error())
			log.Error.Println("Retrying with another PID...")
			peersContacted--
			cancelTimeout()
			continue
		}

		minCpl, maxDistance := getFarthestDistance(currentPeer.String(), queryClosest, false)

		minCplResponse = append(minCplResponse, minCpl)
		minCplAverage += float64(minCpl)
		maxDistanceResponse = append(maxDistanceResponse, maxDistance)
		maxDistanceResponseStd.Add(maxDistance)

		log.Info.Printf("  Min CPL: %d", minCpl)
		log.Info.Printf("  Max Distance: %s (%s)", ToSciNotation(maxDistance), maxDistance)
		log.Info.Printf("  Average:")
		log.Info.Printf("    Min CPL: %f (%d)", minCplAverage/float64(peersContacted+1), maxDistanceResponseStd.getCPL())
		log.Info.Printf("    Mean, STD, M + STD: %s, %s, %s",
			ToSciNotation(maxDistanceResponseStd.GetAverage(Mean)),
			ToSciNotation(maxDistanceResponseStd.GetStdDevAsInt(Mean)),
			ToSciNotation(maxDistanceResponseStd.GetAverage(MeanStdDev)))
		log.Info.Printf("    Weighted Mean, STD, WM + STD : %s, %s, %s",
			ToSciNotation(maxDistanceResponseStd.GetAverage(WeightedMean)),
			ToSciNotation(maxDistanceResponseStd.GetStdDevAsInt(WeightedMean)),
			ToSciNotation(maxDistanceResponseStd.GetAverage(WeightedMeanStdDev)))

		cancelTimeout()
	}

	maxDistanceAverage := big.NewInt(0)
	// Finding the minCpl average
	for i := 0; i < nodesToContact; i++ {
		maxDistanceAverage = maxDistanceAverage.Add(maxDistanceAverage, maxDistanceResponse[i])
	}

	minCplAverage = math.Round(minCplAverage / float64(len(minCplResponse)))
	maxDistanceAverage = maxDistanceAverage.Div(maxDistanceAverage, big.NewInt(int64(len(maxDistanceResponse))))

	return *maxDistanceResponseStd, nil
}
