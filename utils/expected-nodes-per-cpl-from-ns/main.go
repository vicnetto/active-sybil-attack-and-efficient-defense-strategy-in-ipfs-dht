package main

import (
	"flag"
	"fmt"
	detection "github.com/ssrivatsan97/go-libp2p-kad-dht/eclipse-detection"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/probability"
	"os"
	"strconv"
)

type FlagConfig struct {
	networkSize int
	maxCpl      int
}

func help() func() {
	return func() {
		fmt.Println("Usage:", os.Args[0], "[flags]:")
		fmt.Println("    -networkSize  -- Network size to estimate the distribution.")
		fmt.Println("    -maxCpl       -- MaxCPL to show (max: 30).")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flag.IntVar(&flagConfig.networkSize, "networkSize", -1, "")
	flag.IntVar(&flagConfig.maxCpl, "maxCpl", -1, "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if flagConfig.networkSize == -1 {
		fmt.Println("error: flag networkSize missing.")
		missingFlag = true
	}

	if flagConfig.maxCpl == -1 {
		fmt.Println("error: flag maxCpl missing.")
		missingFlag = true
	}

	if missingFlag {
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	if flagConfig.maxCpl > probability.MaxCpl {
		flagConfig.maxCpl = probability.MaxCpl
	}

	return &flagConfig
}

func main() {
	flagConfig := treatFlags()

	detector := detection.New(20)
	distribution := detector.UpdateIdealDistFromNetsize(flagConfig.networkSize)
	kl := probability.GetAllPartialKl(distribution)
	probability.PrintPartialKl(kl, flagConfig.maxCpl)

	for cpl := range distribution {
		if cpl != flagConfig.maxCpl+1 {
			result := strconv.FormatFloat(distribution[cpl], 'f', -1, 64)
			fmt.Printf("cplProbability[%d] = %s\n", cpl, result)
		} else {
			break
		}
	}
}
