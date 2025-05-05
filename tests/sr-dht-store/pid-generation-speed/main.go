package main

import (
	"flag"
	"fmt"
	"github.com/vicnetto/active-sybil-attack/utils/pid-generation/generate"
	"os"
	"time"
)

type FlagConfig struct {
	minCpl         int
	maxCpl         int
	quantityPerCpl int
}

func help() func() {
	return func() {
		fmt.Println("Usage of", os.Args[0], "[flags]:")
		fmt.Println("  -minCpl <int>         -- Minimum CPL of the peers.")
		fmt.Println("  -maxCpl <int>         -- Maximum CPL of the peers.")
		fmt.Println("  -quantityPerCpl <int> -- Quantity of peers to generate per CPL. (default: 20)")
	}
}

func treatFlags() FlagConfig {
	flagConfig := FlagConfig{}
	flag.Usage = help()

	flag.IntVar(&flagConfig.minCpl, "minCpl", 0, "")
	flag.IntVar(&flagConfig.maxCpl, "maxCpl", 0, "")
	flag.IntVar(&flagConfig.quantityPerCpl, "quantityPerCpl", 20, "")
	flag.Parse()

	missingFlag := false

	if flagConfig.minCpl <= 0 {
		fmt.Println("error: flag maxCpl missing.")
		missingFlag = true
	}

	if flagConfig.minCpl <= 0 {
		fmt.Println("error: flag minCpl missing.")
		missingFlag = true
	}

	if missingFlag {
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	return flagConfig
}

func main() {
	flagConfig := treatFlags()

	randomCid := "QmYeE52KHBfBrgXKnFpKQdA1f918xYiMj2VwdUWxdkJLCm"

	var pidGenerateConfig generate.PidGenerateConfig
	pidGenerateConfig.Cid = randomCid
	pidGenerateConfig.Quantity = flagConfig.quantityPerCpl
	pidGenerateConfig.UseAllCpus = true
	pidGenerateConfig.ByCpl = true

	results := make(map[int]time.Duration)
	for cpl := 8; cpl <= 12; cpl++ {
		pidGenerateConfig.Cpl = cpl

		start := time.Now()

		_, _, err := generate.GeneratePeers(pidGenerateConfig, 20, []string{})
		if err != nil {
			panic(err)
		}

		results[cpl] = time.Since(start)
	}

	fmt.Println("Results)")
	fmt.Printf("cpl;timeRequiredTo%dNodes\n", flagConfig.quantityPerCpl)
	for cpl, duration := range results {
		fmt.Printf("%d;%s\n", cpl, duration/3)
	}
}
