package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ipfs/kubo/core"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	kspace "github.com/libp2p/go-libp2p-kbucket/keyspace"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-base32"
	mh "github.com/multiformats/go-multihash"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"math"
	"math/rand"
	"os"
	"slices"
	"sort"
	"time"
)

var log = logger.InitializeLogger()

type ModeType int

const (
	Lookup ModeType = iota
	Query
)

type FlagConfig struct {
	quantity   int
	port       int
	privateKey string
}

type Result struct {
	samePeersInBothClosestList int
	sameMinCpl                 bool
	minCplDifference           int
	minCplQuery                int
	minCplLookup               int
}

func help() func() {
	return func() {
		fmt.Println("Usage:", os.Args[0], "[flags]:")
		fmt.Println("  -quantity <int>  -- Number of tests")
		fmt.Println("  -port <int>          -- Port of the IPFS node. (default: any valid port)")
		fmt.Println("  -privateKey <string> -- Private key of the IPFS node. (default: random node)")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flag.IntVar(&flagConfig.quantity, "quantity", 0, "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if flagConfig.quantity == 0 {
		log.Error.Println("error: flag quantity missing.")
		missingFlag = true
	}

	if missingFlag {
		log.Error.Println()
		flag.Usage()
		os.Exit(1)
	}

	return &flagConfig
}

func queryPeerForClosestFromItself(ctx context.Context, clientNode *core.IpfsNode, pid peer.ID) ([]peer.ID, error) {
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
	log.Info.Println("Random PID)", randomId)

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

func printClosestByOrder(target string, closest []peer.ID) int {
	targetCIDByte, _ := mh.FromB58String(target)
	targetCIDKey := kspace.XORKeySpace.Key(targetCIDByte)

	minCpl := 255
	log.Info.Printf("Closest from %s: [", target)
	for i, node := range closest {
		peerMultiHash, _ := mh.FromB58String(node.String())
		peerKey := kspace.XORKeySpace.Key(peerMultiHash)

		cpl := kbucket.CommonPrefixLen(targetCIDKey.Bytes, peerKey.Bytes)
		if cpl < minCpl {
			minCpl = cpl
		}
		log.Info.Printf(" {%d, (CPL: %d) %s (%s)}", i+1, cpl, node.String(), base32.RawStdEncoding.EncodeToString([]byte(node)))

	}
	log.Info.Println("]")
	fmt.Println()

	return minCpl
}

func closestListDifference(x []peer.ID, y []peer.ID) int {
	var same int

	for _, xx := range x {
		for _, yy := range y {
			if xx == yy {
				same++
				break
			}
		}
	}

	return same
}

func getMedianFromResult(result []Result, modeType ModeType) (median int) {
	var orderedMinCpl []int
	for _, value := range result {
		if modeType == Lookup {
			orderedMinCpl = append(orderedMinCpl, value.minCplLookup)
		} else {
			orderedMinCpl = append(orderedMinCpl, value.minCplQuery)
		}
	}

	sort.Ints(orderedMinCpl)
	size := len(orderedMinCpl)

	if len(orderedMinCpl)%2 == 0 {
		median = (orderedMinCpl[size/2-1] + orderedMinCpl[size/2]) / 2
	} else {
		median = orderedMinCpl[size/2]
	}

	return
}

func getModeFromResult(result []Result, modeType ModeType) (mode int) {
	countMap := make(map[int]int)
	for _, value := range result {
		if modeType == Lookup {
			countMap[value.minCplLookup]++
		} else {
			countMap[value.minCplQuery]++
		}
	}

	//	Find the smallest item with greatest number of occurance in
	//	the input slice
	maxValue := 0
	for _, value := range result {
		var freq, key int
		if modeType == Lookup {
			key = value.minCplLookup
			freq = countMap[key]
		} else {
			key = value.minCplQuery
			freq = countMap[key]
		}

		if freq > maxValue {
			mode = key
			maxValue = freq
		}
	}

	return
}

func main() {
	flagConfig := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())
	var peerConfig ipfspeer.Config
	if len(flagConfig.privateKey) != 0 {
		peerConfig = ipfspeer.ConfigForSpecificNode(flagConfig.port, flagConfig.privateKey)
	} else {
		peerConfig = ipfspeer.ConfigForRandomNode(flagConfig.port)
	}

	_, node, err := ipfspeer.SpawnEphemeral(ctx, peerConfig)
	if err != nil {
		log.Error.Println("Error instantiating the node:", err.Error())
		panic(err)
	}
	defer node.Close()
	log.Info.Println("Peer is up:", node.Identity.String())

	log.Info.Println("Sleep for 10 seconds before starting...")
	time.Sleep(10 * time.Second)
	fmt.Println()

	var alreadyAsked []peer.ID

	var result []Result
	for i := 0; i < flagConfig.quantity; i++ {
		var currentPeer peer.ID
		for {
			peersInRT := node.DHT.WAN.RoutingTable().ListPeers()
			randomPosition := rand.Intn(len(peersInRT))
			currentPeer = peersInRT[randomPosition]

			if slices.Contains(alreadyAsked, currentPeer) {
				continue
			}

			alreadyAsked = append(alreadyAsked, currentPeer)
			break
		}

		log.Info.Printf("%d) Comparing results when searching for: %s", i+1, currentPeer.String())

		log.Info.Printf("%d) Lookup) Getting closest peers to %s...", i+1, currentPeer.String())
		currentPeerMh, _ := mh.FromB58String(currentPeer.String())

		ctxTimeout, cancelTimeout := context.WithTimeout(ctx, 3*time.Minute)
		lookupClosest, err := node.DHT.WAN.GetClosestPeers(ctxTimeout, string(currentPeerMh))
		if err != nil {
			log.Error.Println("Error while asking the closest peers to the CID:", err)
			log.Error.Println("Retrying with another PID...")
			cancelTimeout()
			i--
			continue
		}
		cancelTimeout()

		lookupMinCpl := printClosestByOrder(currentPeer.String(), lookupClosest)

		log.Info.Printf("%d) Query) Getting closest peers to %s...", i+1, currentPeer.String())

		ctxTimeout, cancelTimeout = context.WithTimeout(ctx, 3*time.Minute)
		queryClosest, err := queryPeerForClosestFromItself(ctxTimeout, node, currentPeer)
		if err != nil {
			log.Error.Println("Error while querying the peer:", err.Error())
			log.Error.Println("Retrying with another PID...")
			i--
			continue
		}
		cancelTimeout()
		queryMinCpl := printClosestByOrder(currentPeer.String(), queryClosest)

		same := closestListDifference(queryClosest, lookupClosest)

		var sameMinCpl bool
		if lookupMinCpl == queryMinCpl {
			sameMinCpl = true
		}

		r := Result{
			samePeersInBothClosestList: same,
			sameMinCpl:                 sameMinCpl,
			minCplDifference:           lookupMinCpl - queryMinCpl,
			minCplLookup:               lookupMinCpl,
			minCplQuery:                queryMinCpl,
		}

		log.Info.Printf("For %s)", currentPeer)
		log.Info.Println("  Same peers in both queries:", same)
		log.Info.Printf("  Min CPL (Lookup | Query): %d | %d", r.minCplLookup, r.minCplQuery)
		log.Info.Printf("  Mode Min CPL (Lookup | Query): %d | %d", r.minCplLookup, r.minCplQuery)
		log.Info.Printf("  Same minimum CPL: %t", r.sameMinCpl)
		log.Info.Printf("  Minimum CPL difference (+ Lookup | - Query): %d", r.minCplDifference)

		result = append(result, r)

		fmt.Println()
	}

	var averageSamePeers, averageCplDifference, sameMinimumCplPercentage, averageQueryMinCpl, averageLookupMinCpl float64

	log.Info.Printf("**Results**")
	for i, r := range result {
		log.Info.Printf("Result %d)", i+1)
		log.Info.Println("  Same peers in both queries:", r.samePeersInBothClosestList)
		averageSamePeers += float64(r.samePeersInBothClosestList)

		log.Info.Printf("  Min CPL (Lookup | Query): %d | %d", r.minCplLookup, r.minCplQuery)
		averageQueryMinCpl += float64(r.minCplQuery)
		averageLookupMinCpl += float64(r.minCplLookup)

		log.Info.Printf("  Same minimum CPL: %t", r.sameMinCpl)
		if r.sameMinCpl == true {
			sameMinimumCplPercentage += 1
		}

		log.Info.Printf("  Minimum CPL difference (+ Lookup | - Query): %d", r.minCplDifference)
		averageCplDifference += float64(r.minCplDifference)
	}

	fmt.Println()
	log.Info.Printf("**Global Results**")
	averageLookupMinCpl /= float64(len(result))
	averageQueryMinCpl /= float64(len(result))
	averageAllMinCpl := (averageLookupMinCpl + averageQueryMinCpl) / 2
	averageSamePeers /= float64(len(result))
	sameMinimumCplPercentage = sameMinimumCplPercentage / float64(len(result)) * 100
	averageCplDifference /= float64(len(result))

	log.Info.Printf("  Average minimum CPL (Lookup  |  Query): %.1f | %.1f",
		averageLookupMinCpl, averageQueryMinCpl)
	log.Info.Printf("  Rounded minimum CPL (Lookup  |  Query): %d | %d",
		int(math.Round(averageLookupMinCpl)), int(math.Round(averageQueryMinCpl)))
	log.Info.Printf("  Mode minimum CPL (Lookup  |  Query): %d | %d)",
		getModeFromResult(result, Lookup), getModeFromResult(result, Query))
	log.Info.Printf("  Median minimum CPL (Lookup  |  Query): %d | %d)",
		getMedianFromResult(result, Lookup), getMedianFromResult(result, Query))
	log.Info.Printf("  Same peers in both queries: %.1f", averageSamePeers)
	log.Info.Printf("  Same minimum CPL: %.0f%%", sameMinimumCplPercentage)
	log.Info.Printf("  Minimum CPL difference (+ Lookup | - Query): %.1f", averageCplDifference)

	fmt.Println()
	log.Info.Printf("**CSV export**")
	log.Info.Println("lookupAverageMinCpl;lookupRoundedMinCpl;lookupModeMinCpl;lookupMedianMinCpl")
	log.Info.Printf("%.2f;%d;%d;%d", averageLookupMinCpl, int(math.Round(averageQueryMinCpl)),
		getModeFromResult(result, Lookup), getMedianFromResult(result, Lookup))
	log.Info.Println("queryAverageMinCpl;queryRoundedMinCpl;queryModeMinCpl;queryMedianMinCpl")
	log.Info.Printf("%.2f;%d;%d;%d", averageQueryMinCpl, int(math.Round(averageQueryMinCpl)),
		getModeFromResult(result, Query), getMedianFromResult(result, Query))
	log.Info.Println("lookupMinCpl;queryMinCpl;lookupModeMinCpl;queryModeMinCpl;lookupMedianMinCpl;queryMedianMinCpl;lookupRoundedMinCpl;samePeers;sameMinimumCpl;cplDifference")
	log.Info.Printf("%.2f;%.2f;%.2f;%.2f;%.2f;%.2f", averageLookupMinCpl, averageQueryMinCpl, averageAllMinCpl,
		averageSamePeers, sameMinimumCplPercentage, averageCplDifference)

	cancel()
}
