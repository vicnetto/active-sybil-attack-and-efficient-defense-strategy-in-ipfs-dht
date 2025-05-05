package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ipfs/kubo/core"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"github.com/vicnetto/active-sybil-attack/utils/k-closest-cpl/cpl"
	"os"
)

var log = logger.InitializeLogger()

type FlagConfig struct {
	tests       int
	networkSize int
	privateKey  string
	port        int
}

type DetectionResult struct {
	kl          float64
	networkSize int32
	isAttack    bool
}

func help() func() {
	return func() {
		fmt.Println("Usage:", os.Args[0], "[flags]:")
		fmt.Println("    -tests       -- Test count")
		fmt.Println("    -networkSize -- Network size")
		fmt.Println("    -privateKey  -- Private key to the node calculating the average (default: random pid)")
		fmt.Println("    -port        -- Port (default: any open port)")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flag.IntVar(&flagConfig.tests, "tests", 0, "")
	flag.IntVar(&flagConfig.networkSize, "networkSize", -1, "")
	flag.IntVar(&flagConfig.port, "port", 0, "")
	flag.StringVar(&flagConfig.privateKey, "privateKey", "", "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if flagConfig.tests == 0 {
		fmt.Println("error: flag tests missing.")
		missingFlag = true
	}

	if flagConfig.networkSize <= 0 {
		fmt.Println("error: flag networkSize missing.")
		missingFlag = true
	}

	if missingFlag {
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	return &flagConfig
}

func eclipseDetection(node *core.IpfsNode, inCpl []int) (float64, bool, error) {
	kl := node.DHT.WAN.Detector.ComputeKLFromCounts(inCpl)
	isAttack := node.DHT.WAN.Detector.DetectFromKL(kl)

	return kl, isAttack, nil
}

func main() {
	flagConfig := treatFlags()

	// Create the context
	ctx, cancel := context.WithCancel(context.Background())

	var results []DetectionResult

	var clientConfig ipfspeer.Config
	if len(flagConfig.privateKey) != 0 {
		clientConfig = ipfspeer.ConfigForSpecificNode(flagConfig.port, flagConfig.privateKey)
	} else {
		clientConfig = ipfspeer.ConfigForRandomNode(flagConfig.port)
	}

	for test := 1; test <= flagConfig.tests; test++ {
		log.Info.Printf("%d) ", test)

		_, clientNode, err := ipfspeer.SpawnEphemeral(ctx, clientConfig)
		if err != nil {
			panic(err)
		}
		clientNode.DHT.WAN.SetProvideRegionSize(20)

		netSize, netSizeErr := clientNode.DHT.WAN.NsEstimator.NetworkSize()
		if netSizeErr != nil {
			err = clientNode.DHT.WAN.GatherNetsizeData(ctx)
			if err != nil {
				fmt.Println("Context error:", err)
				err := clientNode.Close()
				if err != nil {
					return
				}
				test--

				continue
			}

			netSize, netSizeErr = clientNode.DHT.WAN.NsEstimator.NetworkSize()
			if netSizeErr != nil {
				fmt.Println("NetSize error:", netSizeErr)
				err := clientNode.Close()
				if err != nil {
					return
				}
				test--

				continue
			}
		}
		log.Info.Println("Network size:", netSize)

		clientNode.DHT.WAN.Detector.UpdateIdealDistFromNetsize(flagConfig.networkSize)

		cid, closest := cpl.GeneratePidAndGetClosestAsString(ctx, clientNode)
		inCpl := cpl.CountInCpl(cid, closest)

		kl, detection, err := eclipseDetection(clientNode, inCpl)
		if err != nil {
			log.Error.Println(err)
			// cid, inCpl = generateRandomPidAndGetClosest(ctx, clientNode)
			test--
			continue
		}

		log.Info.Println(" Results)")
		log.Info.Println("   KL divergence:", kl)
		log.Info.Println("   Attack detected:", detection)

		results = append(results, DetectionResult{kl: kl, networkSize: netSize, isAttack: detection})
		clientNode.Close()
		fmt.Println()
	}
	fmt.Println()

	fmt.Println("Result CSV)")
	fmt.Println("Test;NetSize;Kl;IsAttack")
	for i, detectionResult := range results {
		fmt.Printf("%d;%d;%f;%t\n", i+1, detectionResult.networkSize, detectionResult.kl, detectionResult.isAttack)
	}

	fmt.Println()

	fmt.Println("Total Result CSV)")
	var klAverage float64
	var netSizeAverage float64
	var isAttack int

	for _, detectionResult := range results {
		klAverage += detectionResult.kl
		netSizeAverage += float64(detectionResult.networkSize)

		if detectionResult.isAttack {
			isAttack += 1
		}
	}
	fmt.Printf("%.0f;%f;%.0f%%\n", netSizeAverage/float64(len(results)), klAverage/float64(len(results)), float64(isAttack)/float64(len(results))*100.0)

	cancel()
}
