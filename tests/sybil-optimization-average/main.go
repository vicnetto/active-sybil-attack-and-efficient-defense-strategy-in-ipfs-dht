package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ipfs/kubo/core"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"github.com/vicnetto/active-sybil-attack/utils/k-closest-cpl/cpl"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/optimization"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/probability"
	"os"
)

const KeySize = 256

var log = logger.InitializeLogger()

type FlagConfig struct {
	tests          int
	maxKl          float64
	closestIsSybil bool
	port           int
	privateKey     string
}

func help() func() {
	return func() {
		fmt.Println("Usage of", os.Args[0], "[flags]:")
		fmt.Println("  -tests <int>         -- Number of tests")
		fmt.Println("  -maxKl <float>       -- Optimization max KL (default: 0.85)")
		fmt.Println("  -closestIsSybil      -- Closest node must be a Sybil (default: false)")
		fmt.Println("  -port <int>          -- Port of the IPFS node. (default: any valid port)")
		fmt.Println("  -privateKey <string> -- Private key of the IPFS node. (default: random node)")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}
	flag.Usage = help()

	flag.IntVar(&flagConfig.tests, "tests", 0, "")
	flag.Float64Var(&flagConfig.maxKl, "maxKl", 0.85, "")
	flag.BoolVar(&flagConfig.closestIsSybil, "closestIsSybil", false, "")
	flag.IntVar(&flagConfig.port, "port", 0, "")
	flag.StringVar(&flagConfig.privateKey, "privateKey", "", "")
	flag.Parse()

	missingFlag := false

	if flagConfig.tests == 0 {
		fmt.Println("error: flag tests missing.")
		missingFlag = true
	}

	if missingFlag {
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	return &flagConfig
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

func configureOptimization(inCpl []int, networkSize int, config FlagConfig) (optimization.Config, error) {
	log.Info.Println("Positioning sybils in the following distribution:")
	PrintUsefulCpl(inCpl)

	optimizationConfig, err := optimization.DefaultConfig(inCpl)
	if err != nil {
		return optimization.Config{}, err
	}

	optimizationConfig.MaxKl = config.maxKl
	optimizationConfig.ClosestNodeIsSybil = config.closestIsSybil
	optimizationConfig.NetworkSize = networkSize

	return optimizationConfig, nil
}

func optimizeSybilPositioning(config optimization.Config) (optimization.Result, int, error) {
	positionOptimization, err := optimization.BeginSybilPositionOptimization(config)
	if err != nil {
		log.Error.Println(err)
		return optimization.Result{}, 0, err
	}

	log.Info.Println("Sybils positioned:")
	PrintUsefulCpl(positionOptimization[0].SybilsPerCpl)
	fmt.Println()

	sybils := 0
	for _, quantity := range positionOptimization[0].SybilsPerCpl {
		sybils += quantity
	}

	return positionOptimization[0], sybils, nil
}

func estimateNetworkSize(ctx context.Context, node *core.IpfsNode) (int32, error) {
	var networkSize int32
	var networkSizeErr error

	for {
		networkSize, networkSizeErr = node.DHT.WAN.NsEstimator.NetworkSize()

		if networkSizeErr != nil {
			err := node.DHT.WAN.GatherNetsizeData(ctx)
			if err != nil {
				log.Error.Printf("  %s.. retrying!", err)
				node.Close()
				return 0, err
			}

			networkSize, networkSizeErr = node.DHT.WAN.NsEstimator.NetworkSize()
			if networkSizeErr != nil {
				log.Error.Println("Network Size Error:", networkSizeErr)
				node.Close()
				return 0, err
			}
		}

		break
	}

	log.Info.Println("Network size:", networkSize)
	return networkSize, nil
}

func main() {
	flagConfig := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())

	var lowestSybil, highestSybil int
	lowestSybil = probability.K

	var score, kl, initialKl []float64
	var scoreSum, klSum, initialKlSum float64

	var sybil, networkSize []int
	var sybilSum, networkSizeSum int

	var closestIsSybil []bool
	var closestIsSybilSum int

	var allNodesPerCpl [KeySize]int

	log.Info.Printf("Running %d optimizations:\n", flagConfig.tests)

	for i := 1; i <= flagConfig.tests; i++ {
		log.Info.Printf("%d) Test %d\n", i, i)

		var peerConfig ipfspeer.Config
		if len(flagConfig.privateKey) != 0 {
			peerConfig = ipfspeer.ConfigForSpecificNode(flagConfig.port, flagConfig.privateKey)
		} else {
			peerConfig = ipfspeer.ConfigForRandomNode(flagConfig.port)
		}

		_, node, err := ipfspeer.SpawnEphemeral(ctx, peerConfig)
		if err != nil {
			panic(err)
		}
		log.Info.Println("Peer is UP:", node.Identity.String())

		netSize, err := estimateNetworkSize(ctx, node)
		if err != nil {
			i--
			continue
		}
		node.DHT.WAN.Detector.UpdateIdealDistFromNetsize(int(netSize))

		cid, closestAsString := cpl.GeneratePidAndGetClosestAsString(ctx, node)
		perCpl := cpl.CountInCpl(cid, closestAsString)

		klBeforeOptimization := node.DHT.WAN.Detector.ComputeKLFromCounts(perCpl)
		log.Info.Printf("Initial distribution KL: %f", klBeforeOptimization)

		nodes := 0
		for i := 0; i < KeySize; i++ {
			nodes += perCpl[i]
			allNodesPerCpl[i] += perCpl[i]

			if nodes == 20 {
				break
			}
		}

		optimizationConfig, err := configureOptimization(perCpl, int(netSize), *flagConfig)
		if err != nil {
			panic(err)
		}

		result, sybils, err := optimizeSybilPositioning(optimizationConfig)
		if err != nil {
			log.Error.Println(err)
			i--
			continue
		}

		closestIsSybilBool := isClosestIsASybil(result)

		networkSize = append(networkSize, int(netSize))
		score = append(score, result.Score)
		kl = append(kl, result.Kl)
		initialKl = append(initialKl, klBeforeOptimization)
		sybil = append(sybil, sybils)
		closestIsSybil = append(closestIsSybil, closestIsSybilBool)

		scoreSum += score[i-1]
		klSum += kl[i-1]
		initialKlSum += initialKl[i-1]
		sybilSum += sybil[i-1]
		networkSizeSum += networkSize[i-1]
		if closestIsSybilBool {
			closestIsSybilSum += 1
		}

		if sybils < lowestSybil {
			lowestSybil = sybils
		}

		if sybils > highestSybil {
			highestSybil = sybils
		}

		log.Info.Printf("Score (Score mean): %f (%f)\n", score[i-1], scoreSum/float64(i))
		log.Info.Printf("Initial Kl (Initial Kl mean): %f (%f)\n", initialKl[i-1], initialKlSum/float64(i))
		log.Info.Printf("Kl (Kl mean): %f (%f)\n", kl[i-1], klSum/float64(i))
		log.Info.Printf("Sybils (Sybils mean): %d (%f)\n", sybil[i-1], float64(sybilSum)/float64(i))
		log.Info.Printf("Network Size (NS mean): %d (%f)\n", networkSize[i-1], float64(networkSizeSum)/float64(i))
		log.Info.Printf("Closest is sybil (CIS mean): %t (%d%%)\n", closestIsSybilBool,
			int(float64(closestIsSybilSum)/float64(i)*100))

		if err := node.Close(); err != nil {
			panic(err)
		}
	}

	fmt.Println()

	log.Info.Printf("> Final results in %d tests:\n", flagConfig.tests)
	log.Info.Printf("Score mean: %f\n", scoreSum/float64(flagConfig.tests))

	log.Info.Printf("Initial Kl mean: %f\n", initialKlSum/float64(flagConfig.tests))
	log.Info.Printf("Kl mean: %f\n", klSum/float64(flagConfig.tests))
	log.Info.Printf("Closest is sybil percentage: %d%%\n", int(float64(closestIsSybilSum)/float64(flagConfig.tests)*100))

	PrintUsefulCpl(allNodesPerCpl)
	log.Info.Printf("*- Total nodes: %d\n", flagConfig.tests*20)

	log.Info.Printf("Sybils mean: %f\n", float64(sybilSum)/float64(flagConfig.tests))
	log.Info.Printf("Network Size mean: %f\n", float64(networkSizeSum)/float64(flagConfig.tests))
	log.Info.Printf("Highest sybil quantity: %d\n", highestSybil)
	log.Info.Printf("Lowest sybil quantity: %d\n\n", lowestSybil)

	fmt.Printf("Score;InitialKL;KL;Sybil;NetworkSize;ClosestIsSybil\n")
	printArraysAsCsv(score, initialKl, kl, sybil, networkSize, closestIsSybil)

	cancel()
}
