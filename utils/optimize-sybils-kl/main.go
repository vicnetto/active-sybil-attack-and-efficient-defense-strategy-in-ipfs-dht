package main

import (
	"flag"
	"fmt"
	"optimization-sybils/optimization"
	"optimization-sybils/probability"
	"os"
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
	flag.Usage = help()

	for i := 0; i < probability.MaxCplProbabilitySize; i++ {
		flag.IntVar(&flagConfig.NodesPerCpl[i], strconv.Itoa(i), 0, "")
	}
	flag.IntVar(&flagConfig.Top, "top", 5, "")
	flag.Float64Var(&flagConfig.MaxKl, "maxKl", 0.94, "")
	flag.Float64Var(&flagConfig.MinScore, "minScore", -1, "")
	flag.IntVar(&flagConfig.MinSybils, "minSybils", -1, "")
	flag.Parse()

	missingFlag := false

	var countAllNodes int
	for i := 0; i < probability.MaxCplProbabilitySize; i++ {
		countAllNodes += flagConfig.NodesPerCpl[i]
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
	optimization.Flags = treatFlags()

	probabilities := probability.GetCplProbability()

	optimization.Kl = probability.GetAllPartialKl(probabilities)
	probability.PrintPartialKl(optimization.Kl)

	_, err := optimization.BeginSybilPositionOptimization()
	if err != nil {
		fmt.Println(err)
		return
	}
}
