package main

import (
	"context"
	"flag"
	"fmt"
	gocid "github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/core"
	"github.com/libp2p/go-libp2p/core/peer"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"github.com/vicnetto/active-sybil-attack/utils/k-closest-cpl/cpl"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/optimization"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/probability"
	"os"
	"time"
)

const KeySize = 256

type FlagConfig struct {
	tests int
}

func isClosestIsASybil(result optimization.Result) bool {
	var maxCplSybil, maxCplNodes int

	for i := 0; i < len(result.SybilsPerCpl); i++ {
		if result.SybilsPerCpl[i] != 0 {
			maxCplSybil = i
		}

		if result.NodesPerCpl[i] != 0 {
			maxCplNodes = i
		}
	}

	if maxCplSybil >= maxCplNodes {
		return true
	} else {
		return false
	}
}

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

func optimizeSybilPositioning(inCpl []int) (optimization.Result, int, error) {
	optimizationConfig, err := optimization.DefaultConfig(inCpl)
	if err != nil {
		fmt.Println(err)
		return optimization.Result{}, 0, err
	}

	optimizationConfig.MaxKl = 0.75

	fmt.Println("\nPositioning sybils in the following distribution:")
	PrintUsefulCpl(optimizationConfig.NodesPerCpl)

	positionOptimization, err := optimization.BeginSybilPositionOptimization(optimizationConfig)
	if err != nil {
		fmt.Println(err)
		return optimization.Result{}, 0, err
	}

	fmt.Println("\nSybils positioned:")
	PrintUsefulCpl(positionOptimization[0].SybilsPerCpl)
	fmt.Println()

	sybils := 0
	for _, quantity := range positionOptimization[0].SybilsPerCpl {
		sybils += quantity
	}

	return positionOptimization[0], sybils, nil
}

func help() func() {
	return func() {
		fmt.Println("Usage of", os.Args[0], "[flags]:")
		fmt.Println("	-tests <int>  -- Number of tests (default: 5)")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}
	flag.Usage = help()

	flag.IntVar(&flagConfig.tests, "tests", 5, "")
	flag.Parse()

	return &flagConfig
}

func main() {
	flagConfig := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())

	config := ipfspeer.ConfigForNormalClient(8080)
	_, node, err := ipfspeer.SpawnEphemeral(ctx, config)
	if err != nil {
		panic(err)
	}

	var lowestSybil, highestSybil int
	lowestSybil = probability.K

	var score, kl []float64
	var scoreSum, klSum float64

	var sybil []int
	var sybilSum int

	var closestIsSybil []bool
	var closestIsSybilSum int

	var allNodesPerCpl [KeySize]int

	fmt.Printf("Running %d tests:\n", flagConfig.tests)

	fmt.Println()
	for i := 1; i <= flagConfig.tests; i++ {
		fmt.Printf("%d)\n", i)

		decode, peersAsString := generatePidAndGetClosest(ctx, node)
		inCpl := cpl.CountInCpl(decode, peersAsString)

		nodes := 0
		for i := 0; i < KeySize; i++ {
			nodes += inCpl[i]
			allNodesPerCpl[i] += inCpl[i]

			if nodes == 20 {
				break
			}
		}

		result, sybils, err := optimizeSybilPositioning(inCpl)
		if err != nil {
			fmt.Println(err)
			i--
			continue
		}

		closestIsSybilBool := isClosestIsASybil(result)

		score = append(score, result.Score)
		kl = append(kl, result.Kl)
		sybil = append(sybil, sybils)
		closestIsSybil = append(closestIsSybil, closestIsSybilBool)

		scoreSum += score[i-1]
		klSum += kl[i-1]
		sybilSum += sybil[i-1]
		if closestIsSybilBool {
			closestIsSybilSum += 1
		}

		if sybils < lowestSybil {
			lowestSybil = sybils
		}

		if sybils > highestSybil {
			highestSybil = sybils
		}

		fmt.Printf("Score (Score mean): %f (%f)\n", score[i-1], scoreSum/float64(i))
		fmt.Printf("Kl (Kl mean): %f (%f)\n", kl[i-1], klSum/float64(i))
		fmt.Printf("Sybils (Sybils mean): %d (%f)\n", sybil[i-1], float64(sybilSum)/float64(i))
		fmt.Printf("Closest is sybil (CIS mean): %t (%d%%)\n", closestIsSybilBool,
			int(float64(closestIsSybilSum)/float64(i)*100))

		fmt.Println()
	}

	fmt.Printf("> Final results in %d tests:\n", flagConfig.tests)
	fmt.Printf("Score mean: %f\n", scoreSum/float64(flagConfig.tests))

	fmt.Printf("Kl mean: %f\n", klSum/float64(flagConfig.tests))
	fmt.Printf("Closest is sybil percentage: %d%%\n", int(float64(closestIsSybilSum)/float64(flagConfig.tests)*100))

	PrintUsefulCpl(allNodesPerCpl)
	fmt.Printf("*- Total nodes: %d\n", flagConfig.tests*20)

	fmt.Printf("Sybils mean: %f\n", float64(sybilSum)/float64(flagConfig.tests))
	fmt.Printf("Highest sybil quantity: %d\n", highestSybil)
	fmt.Printf("Lowest sybil quantity: %d\n\n", lowestSybil)
	fmt.Printf("Score, KL, Sybil, ClosestIsSybil\n")
	printArraysAsCsv(score, kl, sybil, closestIsSybil)

	fmt.Printf("\nExiting...\n")

	cancel()

	if err := node.Close(); err != nil {
		panic(err)
	}
}
