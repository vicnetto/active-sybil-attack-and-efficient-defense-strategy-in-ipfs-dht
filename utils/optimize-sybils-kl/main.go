package main

import (
	"flag"
	"fmt"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/optimization"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/probability"
	"os"
	"strconv"
)

func help() func() {
	return func() {
		fmt.Println("Usage of", os.Args[0], "[flags]:")
		fmt.Printf("    -[0-%d] <int>               -- Number of nodes in a specific CPL\n", probability.MaxCpl-1)
		fmt.Println("    -top <int>                  -- Number of top results to display (default: 5)")
		fmt.Printf("    -networkSize <int>          -- Network size to estimate the distribution (default: %d)\n", probability.DefaultNetworkSize)
		fmt.Printf("    -maxKl <float>              -- Maximum KL value of the result (default/max: %f)\n", probability.KlThreshold)
		fmt.Println("    -minKl <float>              -- Minimum KL value of the result (default: -1)")
		fmt.Println("    -minScore <float>           -- Minimum score (default: -1)")
		fmt.Println("    -minSybils <int>            -- Minimum number of sybils (default: -1)")
		fmt.Println("    -closestNodeIsSybil <bool>  -- Closest node to CID must be a sybil (default: false)")
		fmt.Println("    -score <string>             -- Priority when calculating the score. (default: distribution)")
		fmt.Println("                                   Options: quantity, distribution, proximity.")
	}
}

func treatFlags() *optimization.Config {
	flagConfig := optimization.Config{}
	flag.Usage = help()

	for i := 0; i < probability.MaxCpl; i++ {
		flag.IntVar(&flagConfig.NodesPerCpl[i], strconv.Itoa(i), 0, "")
	}

	var scorePriorityAsString string
	flag.IntVar(&flagConfig.Top, "top", 5, "")
	flag.Float64Var(&flagConfig.MaxKl, "maxKl", 0.94, "")
	flag.Float64Var(&flagConfig.MinKl, "minKl", -1, "")
	flag.Float64Var(&flagConfig.MinScore, "minScore", -1, "")
	flag.IntVar(&flagConfig.NetworkSize, "networkSize", probability.DefaultNetworkSize, "")
	flag.IntVar(&flagConfig.MinSybils, "minSybils", -1, "")
	flag.BoolVar(&flagConfig.ClosestNodeIsSybil, "closestNodeIsSybil", false, "")
	flag.StringVar(&scorePriorityAsString, "score", "distribution", "")
	flag.Parse()

	missingFlag := false

	var countAllNodes int
	for i := 0; i < probability.MaxCpl; i++ {
		countAllNodes += flagConfig.NodesPerCpl[i]
	}
	if countAllNodes != probability.K {
		fmt.Println("error: wrong quantity of nodes. k =", probability.K)
		missingFlag = true
	}

	if flagConfig.ScorePriority = optimization.GetScorePriorityFromString(scorePriorityAsString); flagConfig.ScorePriority < 0 {
		fmt.Println("error: invalid score type.")
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
	optimizationFlags := treatFlags()

	fmt.Println("Optimizing the sybils in the following peers configuration:\n")
	optimization.PrintUsefulCpl(optimizationFlags.NodesPerCpl, "Initial nodes")
	fmt.Println("\nWith the following rules:")
	fmt.Println("Top:", optimizationFlags.Top)
	fmt.Println("Max Kl:", optimizationFlags.MaxKl)
	fmt.Println("Min Score:", optimizationFlags.MinScore)
	fmt.Println("Min Sybils:", optimizationFlags.MinSybils)
	fmt.Println("Closest Node Is Sybil:", optimizationFlags.ClosestNodeIsSybil)
	fmt.Println("Network Size:", optimizationFlags.NetworkSize)
	fmt.Println("Score Priority:", optimization.GetStringFromScorePriority(optimizationFlags.ScorePriority), "\n")

	top, err := optimization.BeginSybilPositionOptimization(*optimizationFlags)
	if err != nil {
		fmt.Println(err)
		return
	}

	probability.PrintPartialKl(optimization.Kl)

	fmt.Printf("> Top %d results:\n", optimizationFlags.Top)
	for i, score := range top {
		if score.Score != 0 {
			fmt.Printf("\nResult %d)\n", i+1)
			optimization.PrintFullInformation(score)
		}
	}

}
