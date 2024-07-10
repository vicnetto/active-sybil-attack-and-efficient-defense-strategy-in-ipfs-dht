package main

import (
	"flag"
	"fmt"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/optimization"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/probability"
	"os"
	"runtime/pprof"
	"strconv"
)

func help() func() {
	return func() {
		fmt.Println("Usage of", os.Args[0], "[flags]:")
		fmt.Println("	-[0-keySize=256] <int> -- Number of nodes in a specific CPL")
		fmt.Println("	-top <int>             -- Number of top results to display (default: 5)")
		fmt.Println("	-maxKl <float>         -- Maximum KL value of the result (default/max: 0.94)")
		fmt.Println("	-minScore <float>      -- Minimum score (default: -1)")
		fmt.Println("	-minSybils <int>       -- Minimum number of sybils (default: -1)")
	}
}

func treatFlags() *optimization.Config {
	flagConfig := optimization.Config{}
	flagConfig.NodesPerCplMap = map[int]optimization.CplInformation{}
	flag.Usage = help()

	nodesPerCplAsArray := [probability.MaxCplProbabilitySize]int{}
	for i := 0; i < probability.MaxCplProbabilitySize; i++ {
		flag.IntVar(&nodesPerCplAsArray[i], strconv.Itoa(i), 0, "")
	}
	flag.IntVar(&flagConfig.Top, "top", 5, "")
	flag.Float64Var(&flagConfig.MaxKl, "maxKl", 0.94, "")
	flag.Float64Var(&flagConfig.MinScore, "minScore", -1, "")
	flag.IntVar(&flagConfig.MinSybils, "minSybils", -1, "")
	flag.BoolVar(&flagConfig.ClosestNodeIsSybil, "closestNodeIsSybil", false, "")
	flag.Parse()

	missingFlag := false

	var countAllNodes int
	for i := 0; i < probability.MaxCplProbabilitySize; i++ {
		if nodesPerCplAsArray[i] != 0 {
			flagConfig.NodesPerCplMap[i] = optimization.CplInformation{Reliable: nodesPerCplAsArray[i]}
		}

		countAllNodes += nodesPerCplAsArray[i]
	}
	if countAllNodes != probability.K {
		fmt.Println("error: wrong quantity of nodes. k =", probability.K)
		missingFlag = true
	}

	if missingFlag {
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	return &flagConfig
}

func main() {
	// Create a CPU profile file
	f, err := os.Create("profile.prof")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Start CPU profiling
	if err := pprof.StartCPUProfile(f); err != nil {
		panic(err)
	}
	defer pprof.StopCPUProfile()

	optimization.Flags = treatFlags()

	probabilities := probability.GetCplProbability()

	optimization.Kl = probability.GetAllPartialKl(probabilities)
	probability.PrintPartialKl(optimization.Kl)

	_, err = optimization.BeginSybilPositionOptimization()
	if err != nil {
		fmt.Println(err)
		return
	}
}
