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

type FlagConfig struct {
	maxQueriedPeers int
	tests           int
	port            int
	privateKey      string
}

type Result struct {
	minCplQuery int
}

func help() func() {
	return func() {
		fmt.Println("Usage:", os.Args[0], "[flags]:")
		fmt.Println("  -maxQueriedPeers <int>  -- Number of tests")
		fmt.Println("  -tests <int>  -- Number of tests")
		fmt.Println("  -port <int>          -- Port of the IPFS node. (default: any valid port)")
		fmt.Println("  -privateKey <string> -- Private key of the IPFS node. (default: random node)")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flag.IntVar(&flagConfig.maxQueriedPeers, "maxQueriedPeers", 0, "")
	flag.IntVar(&flagConfig.tests, "tests", 0, "")
	flag.IntVar(&flagConfig.port, "port", 0, "")
	flag.StringVar(&flagConfig.privateKey, "privateKey", "", "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if flagConfig.maxQueriedPeers == 0 {
		log.Error.Println("error: flag maxQueriedPeers missing.")
		missingFlag = true
	}

	if flagConfig.tests == 0 {
		log.Error.Println("error: flag tests missing.")
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

func getMinCpl(target string, closest []peer.ID, print bool) int {
	targetCIDByte, _ := mh.FromB58String(target)
	targetCIDKey := kspace.XORKeySpace.Key(targetCIDByte)

	minCpl := 255
	if print {
		log.Info.Printf("Closest from %s: [", target)
	}
	for i, node := range closest {
		peerMultiHash, _ := mh.FromB58String(node.String())
		peerKey := kspace.XORKeySpace.Key(peerMultiHash)

		cpl := kbucket.CommonPrefixLen(targetCIDKey.Bytes, peerKey.Bytes)
		if cpl < minCpl {
			minCpl = cpl
		}

		if print {
			log.Info.Printf(" {%d, (CPL: %d) %s (%s)}", i+1, cpl, node.String(), base32.RawStdEncoding.EncodeToString([]byte(node)))
		}
	}
	if print {
		log.Info.Println("]")
		fmt.Println()
	}

	return minCpl
}

func getMedianFromResult(result []Result) (median int) {
	var orderedMinCpl []int
	for _, value := range result {
		orderedMinCpl = append(orderedMinCpl, value.minCplQuery)
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

func getModeFromResult(result []Result) (mode int) {
	countMap := make(map[int]int)
	for _, value := range result {
		countMap[value.minCplQuery]++
	}

	//	Find the smallest item with greatest number of occurance in
	//	the input slice
	maxValue := 0
	for _, value := range result {
		var freq, key int
		key = value.minCplQuery
		freq = countMap[key]

		if freq > maxValue {
			mode = key
			maxValue = freq
		}
	}

	return
}

func removePeersFromList(base []peer.ID, remove []peer.ID) []peer.ID {
	for _, r := range remove {
		if index := slices.Index(base, r); index != -1 {
			base = append(base[:index], base[index+1:]...)
		}
	}

	return base
}

func main() {
	flagConfig := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())

	result := make([][][]Result, flagConfig.maxQueriedPeers)
	for i := range result {
		result[i] = make([][]Result, flagConfig.tests)

		for j := range result[i] {
			result[i][j] = make([]Result, i+1)
		}
	}

	var peerConfig ipfspeer.Config
	if len(flagConfig.privateKey) != 0 {
		peerConfig = ipfspeer.ConfigForSpecificNode(flagConfig.port, flagConfig.privateKey)
	} else {
		peerConfig = ipfspeer.ConfigForRandomNode(flagConfig.port)
	}

	for peerQuantity := 0; peerQuantity < flagConfig.maxQueriedPeers; peerQuantity++ {
		_, node, err := ipfspeer.SpawnEphemeral(ctx, peerConfig)
		if err != nil {
			log.Error.Println("Error instantiating the node:", err.Error())
			panic(err)
		}

		log.Info.Println("Peer is up:", node.Identity.String())

		log.Info.Println("Sleep for 10 seconds before starting...")
		time.Sleep(10 * time.Second)
		fmt.Println()

		var alreadyQueriedPeers []peer.ID

		for currentTest := 0; currentTest < flagConfig.tests; currentTest++ {
			for currentPeerIteration := 0; currentPeerIteration < peerQuantity+1; currentPeerIteration++ {
				var currentPeer peer.ID
				for {
					// Get random peer from the Routing Table
					peersInRT := node.DHT.WAN.RoutingTable().ListPeers()
					validPeersInRT := removePeersFromList(peersInRT, alreadyQueriedPeers)
					if len(validPeersInRT) == 0 {
						log.Info.Printf("Already asked for all the nodes in the RT. Performing random DHT request...")
						ctxTimeout, cancelCtxTimeout := context.WithTimeout(ctx, 10*time.Second)
						id, err := node.DHT.WAN.RoutingTable().GenRandPeerID(0)
						if err != nil {
							log.Error.Println("Error generating random peer:", err)
							panic(err)
						}

						node.DHT.WAN.GetClosestPeers(ctxTimeout, id.String())
						cancelCtxTimeout()
						continue
					}

					if len(validPeersInRT) == 0 {
						continue
					}

					randomPosition := rand.Intn(len(validPeersInRT))
					currentPeer = validPeersInRT[randomPosition]
					if slices.Contains(alreadyQueriedPeers, currentPeer) {
						continue
					}

					break
				}
				alreadyQueriedPeers = append(alreadyQueriedPeers, currentPeer)

				log.Info.Printf("%d %d %d) Querying %s for their closest peers...", peerQuantity+1, currentTest+1,
					currentPeerIteration+1, currentPeer.String())

				ctxTimeout, cancelTimeout := context.WithTimeout(ctx, 3*time.Minute)
				queryClosest, err := queryPeerForClosestFromItself(ctxTimeout, node, currentPeer)
				if err != nil {
					log.Error.Println("Error while querying the peer:", err.Error())
					log.Error.Println("Retrying with another PID...")
					currentPeerIteration--
					continue
				}
				cancelTimeout()
				queryMinCpl := getMinCpl(currentPeer.String(), queryClosest, false)

				// log.Info.Printf("For %s)", currentPeer)
				log.Info.Printf("  Min CPL: %d", queryMinCpl)

				result[peerQuantity][currentTest][currentPeerIteration].minCplQuery = queryMinCpl
			}

			var averageQueryMinCpl float64
			for i := 0; i <= peerQuantity; i++ {
				averageQueryMinCpl += float64(result[peerQuantity][currentTest][i].minCplQuery)
			}
			averageQueryMinCpl /= float64(peerQuantity + 1)

			log.Info.Printf("Result %d %d)", peerQuantity+1, currentTest+1)
			log.Info.Printf("  Average minimum CPL Query: %.1f", averageQueryMinCpl)
			log.Info.Printf("  Rounded minimum CPL Query: %d ", int(math.Round(averageQueryMinCpl)))
			log.Info.Printf("  Mode minimum CPL Query: %d", getModeFromResult(result[peerQuantity][currentTest]))
			log.Info.Printf("  Median minimum CPL Query: %d", getMedianFromResult(result[peerQuantity][currentTest]))
		}

		node.Close()

		fmt.Println()
	}

	log.Info.Printf("**Final Results**")
	fmt.Printf("askedPeers;testNumber;averageMinCplOverTests;roundedAverageMinCplOverTests;averageModeMinCpl;averageMedianMinCpl\n")
	for i, peerQuantity := range result {
		var averageQueryMinCplPerPeerQuantity float64
		// log.Info.Printf("For peer quantity %d)", i+1)

		for j, test := range peerQuantity {
			var averageQueryMinCplPerTest float64
			for _, currentPeer := range test {
				averageQueryMinCplPerTest += float64(currentPeer.minCplQuery)
			}
			averageQueryMinCplPerTest /= float64(len(test))

			fmt.Printf("%d;%d;%.1f;%d;%d;%d\n", i+1, j+1, averageQueryMinCplPerTest, int(math.Round(averageQueryMinCplPerTest)),
				getModeFromResult(test), getMedianFromResult(test))
			averageQueryMinCplPerPeerQuantity += averageQueryMinCplPerTest
		}

		averageQueryMinCplPerPeerQuantity /= float64(len(peerQuantity))
	}

	cancel()
}
