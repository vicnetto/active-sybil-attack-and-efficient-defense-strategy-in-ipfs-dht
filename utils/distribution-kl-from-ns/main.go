package main

import (
	"flag"
	"fmt"
	detection "github.com/ssrivatsan97/go-libp2p-kad-dht/eclipse-detection"
	"github.com/vicnetto/active-sybil-attack/logger"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/probability"
	"os"
	"strconv"
)

const KeySize = 256

var log = logger.InitializeLogger()

type FlagConfig struct {
	nodesPerCpl []int
	networkSize *int
}

func help() func() {
	return func() {
		fmt.Println("Usage:", os.Args[0], "[flags]:")
		fmt.Println("    -networkSize  -- Network size to estimate the distribution")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}
	flagConfig.nodesPerCpl = make([]int, KeySize)

	for i := 0; i < KeySize; i++ {
		flag.IntVar(&flagConfig.nodesPerCpl[i], strconv.Itoa(i), 0, "")
	}

	flagConfig.networkSize = flag.Int("networkSize", -1, "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if *flagConfig.networkSize == -1 {
		fmt.Println("error: flag networkSize missing.")
		missingFlag = true
	}

	var countAllNodes int
	for i := 0; i < KeySize; i++ {
		countAllNodes += flagConfig.nodesPerCpl[i]
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
	flagConfig := treatFlags()

	detector := detection.New(20)
	detector.UpdateIdealDistFromNetsize(*flagConfig.networkSize)
	klValue := detector.ComputeKLFromCounts(flagConfig.nodesPerCpl)
	log.Info.Printf("The KL value is: %.15f", klValue)
}
