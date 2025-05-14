package main

import (
	"context"
	"flag"
	"fmt"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/sr"
	srutils "github.com/libp2p/go-libp2p-kad-dht/sr/utils"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"math/big"
	"os"
	"time"
)

var log = logger.InitializeLogger()

type FlagConfig struct {
	tests      int
	port       int
	privateKey string
}

func help() func() {
	return func() {
		fmt.Println("Usage:", os.Args[0], "[flags]:")
		fmt.Println("  -tests <int>          -- Number of tests to estimate the average alpha")
		fmt.Println("  -port <int>          -- Port of the IPFS node. (default: any valid port)")
		fmt.Println("  -privateKey <string> -- Private key of the IPFS node. (default: random node)")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flag.IntVar(&flagConfig.tests, "tests", 0, "")
	flag.IntVar(&flagConfig.port, "port", 0, "")
	flag.StringVar(&flagConfig.privateKey, "privateKey", "", "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if flagConfig.tests == 0 {
		log.Error.Println("error: flag tests missing.")
		missingFlag = true
	}
	if missingFlag {
		log.Error.Println()
		flag.Usage()
		os.Exit(1)
	}

	return &flagConfig
}

func getAlphaFromErrorSquared(initialMaxDistance []*big.Int, afterMaxDistance []*big.Int) float64 {
	bestValue := 0.5
	step := 1.0
	var upper, lower float64

	for depth := 1; depth <= 3; depth++ {
		errorSquared := make(map[float64]*big.Int)

		step /= 10
		if depth == 1 {
			upper = bestValue + 4*step
			lower = bestValue - 4*step
		} else {
			upper = bestValue + 9*step
			lower = bestValue - 9*step
		}

		log.Info.Printf("Depth %d) upper=%f, lower=%f, step=%f", depth, upper, lower, step)

		for alpha := lower; alpha <= upper; alpha = alpha + step {
			sr.SetParameters(alpha, 0.5)

			beforeAverage := sr.NewWelfordMovingAverage()
			currentAverage := sr.NewWelfordMovingAverage()
			for i, md := range initialMaxDistance {
				currentAverage.Add(md)

				if (i+1)%1 == 0 {
					beforeAverage.Add(currentAverage.GetAverage(sr.Mean))
					currentAverage = sr.NewWelfordMovingAverage()
				}
			}
			if currentAverage.GetAverage(sr.Mean).Cmp(big.NewInt(0)) != 0 {
				beforeAverage.Add(currentAverage.GetAverage(sr.Mean))
			}

			average := sr.NewWelfordMovingAverageFromMean(*beforeAverage)
			currentAverage = sr.NewWelfordMovingAverage()
			for i, md := range afterMaxDistance {
				currentAverage.Add(md)

				if (i+1)%1 == 0 {
					average.Add(currentAverage.GetAverage(sr.Mean))
					currentAverage = sr.NewWelfordMovingAverage()
				}
			}
			if currentAverage.GetAverage(sr.Mean).Cmp(big.NewInt(0)) != 0 {
				average.Add(currentAverage.GetAverage(sr.Mean))
			}

			errorSquared[alpha] = average.GetErrorSquaredAverage()
		}

		var lowestAlpha = lower
		var lowestError = errorSquared[lower]

		for alpha, value := range errorSquared {
			if value.Cmp(lowestError) <= 0 {
				lowestAlpha = alpha
				lowestError = value
			}
		}

		bestValue = lowestAlpha

		log.Info.Printf("Results (depth=%d):", depth)
		for alpha := lower; alpha <= upper; alpha = alpha + step {
			log.Info.Printf("  alpha=%.3f) %s", alpha, dht.ToSciNotation(errorSquared[alpha]))
		}
		log.Info.Printf("Best alpha: %.3f (%s)\n", lowestAlpha, dht.ToSciNotation(lowestError))

	}

	return bestValue
}

func main() {
	flagConfig := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())

	var peerConfig ipfspeer.Config
	if len(flagConfig.privateKey) != 0 {
		peerConfig = ipfspeer.ConfigForSpecificNode(flagConfig.port, flagConfig.privateKey)
	} else {
		peerConfig = ipfspeer.ConfigForRandomNode(flagConfig.port)
	}

	_, clientNode, err := ipfspeer.SpawnEphemeral(ctx, peerConfig)
	if err != nil {
		panic(err)
	}
	log.Info.Println("PID is UP:", clientNode.Identity.String())

	log.Info.Println("Sleep for 10 for initiating the RT...")
	time.Sleep(10 * time.Second)

	var alpha []float64

	var alreadyQueriedPeers []peer.ID
	for i := 1; i <= flagConfig.tests; i++ {
		log.Info.Printf("%d) Test %d", i, i)
		log.Info.Println("Query)")
		var initialMaxDistance []*big.Int
		maxDistance, err := srutils.GetFarthestKByQueryWithAlreadyQueriedPeers(ctx, clientNode, &alreadyQueriedPeers)
		for i := 0; i < 10; i++ {
			if err != nil {
				panic(err)
			}
			initialMaxDistance = append(initialMaxDistance, maxDistance)

			log.Info.Printf("    Max Distance: %s (%s)", dht.ToSciNotation(maxDistance), maxDistance)
		}

		log.Info.Println("Lookups)")
		var afterMaxDistance []*big.Int
		for i := 0; i < 15; i++ {
			maxDistance, err := srutils.GetFarthestKByDbWithAlreadyQueriedPeers("../../../db", &[]peer.ID{})
			if err != nil {
				panic(err)
			}
			afterMaxDistance = append(afterMaxDistance, maxDistance)

			log.Info.Printf("    Max Distance: %s (%s)", dht.ToSciNotation(maxDistance), maxDistance)
		}

		alpha = append(alpha, getAlphaFromErrorSquared(initialMaxDistance, afterMaxDistance))
	}

	var alphaAverage float64
	for _, a := range alpha {
		alphaAverage += a
	}
	alphaAverage /= float64(len(alpha))

	fmt.Println()
	log.Info.Printf("Final alpha: %.3f", alphaAverage)

	clientNode.Close()
	cancel()
}
