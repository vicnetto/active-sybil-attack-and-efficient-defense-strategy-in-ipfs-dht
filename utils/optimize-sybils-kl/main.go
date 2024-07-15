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
		fmt.Printf("	-[0-%d] <int>      -- Number of nodes in a specific CPL\n", probability.MaxCpl-1)
		fmt.Println("	-top <int>         -- Number of top results to display (default: 5)")
		fmt.Println("	-maxKl <float>     -- Maximum KL value of the result (default/max: 0.94)")
		fmt.Println("	-minKl <float>     -- Minimum KL value of the result (default: -1)")
		fmt.Println("	-minScore <float>  -- Minimum score (default: -1)")
		fmt.Println("	-minSybils <int>   -- Minimum number of sybils (default: -1)")
	}
}

func treatFlags() *optimization.Config {
	flagConfig := optimization.Config{}
	flag.Usage = help()

	for i := 0; i < probability.MaxCpl; i++ {
		flag.IntVar(&flagConfig.NodesPerCpl[i], strconv.Itoa(i), 0, "")
	}
	flag.IntVar(&flagConfig.Top, "top", 5, "")
	flag.Float64Var(&flagConfig.MaxKl, "maxKl", 0.94, "")
	flag.Float64Var(&flagConfig.MinKl, "minKl", -1, "")
	flag.Float64Var(&flagConfig.MinScore, "minScore", -1, "")
	flag.IntVar(&flagConfig.MinSybils, "minSybils", -1, "")
	flag.BoolVar(&flagConfig.ClosestNodeIsSybil, "closestNodeIsSybil", false, "")
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

	if missingFlag {
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	return &flagConfig
}

func main() {
	optimizationFlags := treatFlags()

	fmt.Println("Optimizing the sybils in the following peers configuration:")
	optimization.PrintCpl(optimizationFlags.NodesPerCpl)
	fmt.Println("\nWith the following rules:")
	fmt.Println("Top:", optimizationFlags.Top)
	fmt.Println("Max Kl:", optimizationFlags.MaxKl)
	fmt.Println("Min Score:", optimizationFlags.MinScore)
	fmt.Println("Min Sybils:", optimizationFlags.MinSybils)
	fmt.Println("Closest Node Is Sybil:", optimizationFlags.ClosestNodeIsSybil, "\n")

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
