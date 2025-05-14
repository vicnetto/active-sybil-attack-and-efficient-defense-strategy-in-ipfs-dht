package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"github.com/vicnetto/active-sybil-attack/utils/k-closest-cpl/cpl"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/optimization"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/probability"
	"math"
	"os"
)

var log = logger.InitializeLogger()

type FlagConfig struct {
	tests       int
	port        int
	maxKl       float64
	networkSize int
	privateKey  string
}

func help() func() {
	return func() {
		fmt.Println("Usage of", os.Args[0], "[flags]:")
		fmt.Println("  -tests <int>         -- Number of tests (default: 1)")
		fmt.Println("  -privateKey <string> -- Private key of the IPFS node. (default: random node)")
		fmt.Println("  -maxKl <int>         -- Max KL of the generated distribution. (default: 0.85)")
		fmt.Println("  -networkSize <int>   -- Private key of the IPFS node. (default: 13239)")
		fmt.Println("  -port <int>          -- Port of the IPFS node. (default: any valid port)")
	}
}

func treatFlags() FlagConfig {
	flagConfig := FlagConfig{}
	flag.Usage = help()

	flag.IntVar(&flagConfig.tests, "tests", 0, "")
	flag.IntVar(&flagConfig.port, "port", 0, "")
	flag.IntVar(&flagConfig.networkSize, "networkSize", 13239, "")
	flag.Float64Var(&flagConfig.maxKl, "maxKl", 0.85, "")
	flag.StringVar(&flagConfig.privateKey, "privateKey", "", "")
	flag.Parse()

	missingFlag := false

	if flagConfig.tests <= 0 {
		fmt.Println("error: flag tests missing.")
		missingFlag = true
	}

	if missingFlag {
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	return flagConfig
}

type PriorityTestLog struct {
	score, kl                    []float64
	sybil, closerThanAllReliable []int
	closestIsSybil               []bool

	scoreSum, klSum                    float64
	sybilSum, closerThenAllReliableSum int
	closestIsSybilSum                  int

	scoreHigh, scoreLow               float64
	klHigh, klLow                     float64
	sybilHigh, sybilLow               int
	closerSybilsHigh, closerSybilsLow int
}

func generateScorePriorityMapToData() map[optimization.ScorePriority]PriorityTestLog {
	priorityData := map[optimization.ScorePriority]PriorityTestLog{}

	for priority := optimization.Quantity; priority <= optimization.Proximity; priority++ {
		priorityData[priority] = PriorityTestLog{scoreLow: math.MaxFloat64, klLow: math.MaxFloat64,
			sybilLow: math.MaxInt, closerSybilsLow: math.MaxInt}
	}

	return priorityData
}

func isClosestASybil(result optimization.Result) bool {
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

func saveResultIntoResultData(result optimization.Result, sybils int, resultData PriorityTestLog) PriorityTestLog {
	closestIsSybilBool := isClosestASybil(result)

	resultData.score = append(resultData.score, result.Score)
	resultData.scoreSum += result.Score

	resultData.kl = append(resultData.kl, result.Kl)
	resultData.klSum += result.Kl

	resultData.sybil = append(resultData.sybil, sybils)
	resultData.sybilSum += sybils

	var reliableNodes, closerThanAllReliable int
	var closer bool
	for i := 0; i < len(result.NodesPerCpl); i++ {
		reliableNodes += result.NodesPerCpl[i] - result.SybilsPerCpl[i]

		if reliableNodes == (probability.K - sybils) {
			closer = true
		}

		if closer {
			closerThanAllReliable += result.SybilsPerCpl[i]
		}
	}

	resultData.closerThanAllReliable = append(resultData.closerThanAllReliable, closerThanAllReliable)
	resultData.closerThenAllReliableSum += closerThanAllReliable

	resultData.closestIsSybil = append(resultData.closestIsSybil, closestIsSybilBool)
	if closestIsSybilBool {
		resultData.closestIsSybilSum += 1
	}

	if result.Score < resultData.scoreLow {
		resultData.scoreLow = result.Score
	}
	if result.Score > resultData.scoreHigh {
		resultData.scoreHigh = result.Score
	}

	if result.Kl < resultData.klLow {
		resultData.klLow = result.Kl
	}
	if result.Kl > resultData.klHigh {
		resultData.klHigh = result.Kl
	}

	if closerThanAllReliable < resultData.closerSybilsLow {
		resultData.closerSybilsLow = closerThanAllReliable
	}
	if closerThanAllReliable > resultData.closerSybilsHigh {
		resultData.closerSybilsHigh = closerThanAllReliable
	}

	if sybils < resultData.sybilLow {
		resultData.sybilLow = sybils
	}
	if sybils > resultData.sybilHigh {
		resultData.sybilHigh = sybils
	}

	return resultData
}

func configureSybilOptimization(flagConfig FlagConfig, inCpl []int) (optimization.Config, error) {
	optimizationConfig, err := optimization.DefaultConfig(inCpl)
	if err != nil {
		log.Info.Println(err)
		return optimization.Config{}, err
	}

	optimizationConfig.MaxKl = flagConfig.maxKl
	optimizationConfig.NetworkSize = flagConfig.networkSize

	log.Info.Println("Positioning sybils in the following distribution:")
	PrintUsefulCpl(optimizationConfig.NodesPerCpl)

	return optimizationConfig, nil
}

func optimizeSybilPositioning(optimizationConfig optimization.Config) (optimization.Result, int, error) {
	positionOptimization, err := optimization.BeginSybilPositionOptimization(optimizationConfig)
	if err != nil {
		log.Info.Println(err)
		return optimization.Result{}, 0, err
	}

	log.Info.Printf("%s)",
		optimization.GetStringFromScorePriority(optimizationConfig.ScorePriority))
	PrintUsefulCpl(positionOptimization[0].SybilsPerCpl)

	sybils := 0
	for _, quantity := range positionOptimization[0].SybilsPerCpl {
		sybils += quantity
	}

	return positionOptimization[0], sybils, nil
}

func main() {
	flagConfig := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	defer node.Close()

	log.Info.Printf("Peer is up: %s", node.Identity.String())

	priorityData := generateScorePriorityMapToData()
	var allNodesPerCpl [probability.KeySize]int

	log.Info.Printf("Running a total of %d tests:", flagConfig.tests)

	for i := 0; i < flagConfig.tests; i++ {
		log.Info.Printf("%d) Test %d", i+1, i+1)

		cid, peersAsString := cpl.GeneratePidAndGetClosestAsString(ctx, node)
		inCpl := cpl.CountInCpl(cid, peersAsString)

		nodes := 0
		for j := 0; j < probability.KeySize; j++ {
			nodes += inCpl[j]
			allNodesPerCpl[j] += inCpl[j]

			if nodes == 20 {
				break
			}
		}

		sybilOptimization, err := configureSybilOptimization(flagConfig, inCpl)
		if err != nil {
			log.Info.Println(err)
			i--
			continue
		}

		for priority := optimization.Quantity; priority <= optimization.Proximity; priority++ {
			currentPriorityData := priorityData[priority]

			sybilOptimization.ScorePriority = priority
			result, sybils, err := optimizeSybilPositioning(sybilOptimization)
			if err != nil {
				log.Info.Println(err)
				priority--
				continue
			}

			priorityData[priority] = saveResultIntoResultData(result, sybils, currentPriorityData)
			printCurrentStatus(i, priorityData[priority])
		}

		fmt.Println()
	}

	log.Info.Printf("> Final results in %d tests:\n", flagConfig.tests)
	PrintUsefulCpl(allNodesPerCpl)
	log.Info.Printf("  *- Total nodes: %d\n", flagConfig.tests*20)

	for priority := optimization.Quantity; priority <= optimization.Proximity; priority++ {
		log.Info.Printf("%s)", optimization.GetStringFromScorePriority(priority))
		printGlobalStatus(priorityData[priority])
	}

	fmt.Println()
	log.Info.Printf("CSV export:")
	for priority := optimization.Quantity; priority <= optimization.Proximity; priority++ {
		currentPriorityData := priorityData[priority]
		fmt.Printf("%s)\n", optimization.GetStringFromScorePriority(priority))
		fmt.Printf("Score, KL, Sybil, CloserThanAllReliable, ClosestIsSybil\n")
		printArraysAsCsv(currentPriorityData.score, currentPriorityData.kl, currentPriorityData.sybil,
			currentPriorityData.closerThanAllReliable, currentPriorityData.closestIsSybil)
	}
}
