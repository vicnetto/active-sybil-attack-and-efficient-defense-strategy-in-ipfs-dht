package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"github.com/vicnetto/active-sybil-attack/utils/k-closest-cpl/cpl"
	"os"
)

var log = logger.InitializeLogger()

type FlagConfig struct {
	tests *int
}

func help() func() {
	return func() {
		fmt.Println("Usage:", os.Args[0], "[flags]:")
		fmt.Println("    -tests  -- Test count (default: 3)")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flagConfig.tests = flag.Int("tests", 3, "")

	flag.Usage = help()
	flag.Parse()

	return &flagConfig
}

func main() {
	flagConfig := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())

	clientConfig := ipfspeer.ConfigForNormalClient(0)

	_, clientNode, err := ipfspeer.SpawnEphemeral(ctx, clientConfig)
	if err != nil {
		panic(err)
	}
	defer clientNode.Close()

	log.Info.Println("PID is up:", clientNode.Identity.String())
	fmt.Println()

	cplMean := map[int]int{}
	for test := 1; test <= *flagConfig.tests; test++ {
		cid, closest := cpl.GeneratePidAndGetClosestAsString(ctx, clientNode)
		inCpl := cpl.CountInCpl(cid, closest)

		log.Info.Println("Closest k per CPL:")
		for currentCpl, quantity := range inCpl {
			if quantity != 0 {
				log.Info.Printf("%d;%d;%.2f", currentCpl, quantity, float64(quantity*100)/float64(20**flagConfig.tests))
			}
		}

		for currentCpl, quantity := range inCpl {
			if quantity != 0 {
				cplMean[currentCpl] += quantity
			}
		}

		log.Info.Printf("Current Average (total: %d):", test)
		for i := 0; i < 256; i++ {
			if cplMean[i] != 0 {
				log.Info.Printf("%d;%d;%.2f", i, cplMean[i], float64(cplMean[i]*100)/float64(20**flagConfig.tests))
			}
		}
		fmt.Println()
	}

	cancel()
}
