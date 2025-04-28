package main

import (
	"context"
	"flag"
	"fmt"
	gocid "github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/core"
	kb "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"os"
	"strconv"
	"time"
)

const KeySize = 256

var log = logger.InitializeLogger()

type FlagConfig struct {
	quantity *int
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

/*
func PrintUsefulCpl(nodesPerCpl interface{}) {
	var minCpl, maxCpl int
	nodes := reflect.ValueOf(nodesPerCpl)

	if nodes.Kind() == reflect.Slice || nodes.Kind() == reflect.Array {
		var count int64

		for i := 0; i < nodes.Len(); i++ {
			if count == 0 && nodes.Index(i).Int() != 0 {
				minCpl = i
			}

			if nodes.Index(i).Int() != 0 {
				maxCpl = i
			}

			count += nodes.Index(i).Int()
		}

		fmt.Printf("                 ")
		for i := minCpl; i <= maxCpl; i++ {
			fmt.Printf("%4d", i)
		}
		fmt.Println()

		fmt.Printf("Nodes per CPL : ")
		fmt.Printf("[")
		for i := minCpl; i <= maxCpl; i++ {
			fmt.Printf(" %3d", nodes.Index(i).Int())
		}
		fmt.Printf(" ]\n")
	}
}
*/

func generatePidAndGetClosest(ctx context.Context, node *core.IpfsNode) (gocid.Cid, []string) {
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
		fmt.Println("PID:", cid.String())
		fmt.Printf("Getting closest peers to %s...\n", cid.String())

		timeoutCtx, cancelTimeoutCtx := context.WithTimeout(ctx, 30*time.Second)

		// Get the closest peers to verify the CPL of each one
		peers, err = node.DHT.WAN.GetClosestPeers(timeoutCtx, string(cid.Hash()))
		if err != nil {
			fmt.Println(err)
			cancelTimeoutCtx()
			continue
		}

		cancelTimeoutCtx()
		break
	}

	var peersAsString []string
	for _, peerId := range peers {
		peersAsString = append(peersAsString, peerId.String())
	}

	return cid, peersAsString
}

func help() func() {
	return func() {
		fmt.Println("Usage:", os.Args[0], "[flags]:")
		fmt.Println("    -quantity     -- Number of tests (default: 1)")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flagConfig.quantity = flag.Int("quantity", 1, "")

	flag.Usage = help()
	flag.Parse()

	return &flagConfig
}

/*
func configForNormalClient(identitya config.Identity, port int) *PeerConfig {
	// Any IP address
	ip := "0.0.0.0"

	// CID will not be used, neither the sybils list.
	cid := ""
	sybilFilePath := ""

	if len(identity.PrivKey) == 0 && len(identity.PeerID) == 0 {
		return &PeerConfig{port, &ip, &cid, config.Identity{}, &sybilFilePath}
	} else {
		return &PeerConfig{port, &ip, &cid, identity, &sybilFilePath}
	}
}
*/

/*
func generateIdentityAndRandomCid(ctx context.Context, node *core.IpfsNode) (config.Identity, gocid.Cid, [KeySize]int) {
	decode, peersAsString := generatePidAndGetClosest(ctx, node)

	var allNodesPerCpl [KeySize]int
	inCpl := CountInCpl(decode, peersAsString)

	nodes := 0
	for i := 0; i < KeySize; i++ {
		nodes += inCpl[i]
		allNodesPerCpl[i] += inCpl[i]

		if nodes == 20 {
			break
		}
	}

	var identity config.Identity

	if len(identity.PrivKey) == 0 && len(identity.PeerID) == 0 {
		key, err := crypto.MarshalPrivateKey(node.PrivateKey)
		if err != nil {
			panic(err)
		}

		privateMarshaledKey := base64.StdEncoding.EncodeToString(key)

		identity.PrivKey = privateMarshaledKey
		identity.PeerID = node.Identity.String()
	}

	return identity, decode, allNodesPerCpl
}
*/

func main() {
	flagConfig := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())

	var averageCplPercentage []float64
	clientConfig := ipfspeer.ConfigForNormalClient(0)

	for i := 1; i <= *flagConfig.quantity; i++ {
		_, clientNode, err := ipfspeer.SpawnEphemeral(ctx, clientConfig)
		if err != nil {
			panic(err)
		}

		log.Info.Println("PID is up:", clientNode.Identity.String())

		log.Info.Println("Setting provider region size to:", 20)
		clientNode.DHT.WAN.SetProvideRegionSize(20)

		log.Info.Printf("Sleeping for 10 seconds...")
		time.Sleep(10 * time.Second)

		netSize, netSizeError := clientNode.DHT.WAN.NsEstimator.NetworkSize()
		if netSizeError != nil {
			for {
				err = clientNode.DHT.WAN.GatherNetsizeData(ctx)
				if err != nil {
					log.Error.Printf("  %s.. retrying!", err)
					continue
				}
				break
			}

			netSize, netSizeError = clientNode.DHT.WAN.NsEstimator.NetworkSize()
			if netSizeError != nil {
				log.Error.Println("Network size estimator error:", netSizeError)
				return
			}
		}

		log.Info.Println("Estimated network size:", netSize)

		cplPercentage := clientNode.DHT.WAN.Detector.UpdateIdealDistFromNetsize(int(netSize))

		if i == 1 {
			averageCplPercentage = cplPercentage
		} else {
			for j := 0; j < len(averageCplPercentage); j++ {
				averageCplPercentage[j] += cplPercentage[j]
			}
		}

		log.Info.Println("Estimated percentages per CPL:")
		for j := 0; j < 50; j++ {
			averageAsString := strconv.FormatFloat(averageCplPercentage[j]/float64(i), 'f', -1, 64)
			log.Info.Printf("  CPL %d = %s\n", j, averageAsString)
		}

		if err := clientNode.Close(); err != nil {
			panic(err)
		}

		err = clientNode.Close()
		if err != nil {
			return
		}
	}

	cancel()
}
