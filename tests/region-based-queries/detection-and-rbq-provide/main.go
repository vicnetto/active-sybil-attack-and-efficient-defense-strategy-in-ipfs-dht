package main

import (
	"context"
	"flag"
	"fmt"
	gocid "github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"github.com/vicnetto/active-sybil-attack/utils/k-closest-cpl/cpl"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var log = logger.InitializeLogger()

type FlagConfig struct {
	cid  string
	port int
}

func fmtDuration(duration time.Duration) string {
	m := int(duration.Minutes()) % 60
	s := int(duration.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d", m, s)
}

func help() func() {
	return func() {
		fmt.Println("Usage:", os.Args[0], "[flags]:")
		fmt.Println("  -cid <string> -- CID to be published.")
		fmt.Println("  -port <int>   -- Port to instantiate the node. (default: random valid port)")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flag.StringVar(&flagConfig.cid, "cid", "", "")
	flag.IntVar(&flagConfig.port, "port", 0, "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if len(flagConfig.cid) == 0 {
		fmt.Println("error: flag cid missing.")
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

	start := time.Now()
	peerConfig := ipfspeer.ConfigForRandomNode(flagConfig.port)

	cid, err := gocid.Decode(flagConfig.cid)
	if err != nil {
		panic("Failed to decode the identifier passed as parameter.. send a valid CID.")
	}
	log.Info.Println("CID:", cid)
	log.Info.Println("PID:", peerConfig.Identity.PeerID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, clientNode, err := ipfspeer.SpawnEphemeral(ctx, peerConfig)
	if err != nil {
		panic(err)
	}
	log.Info.Println("Peer is UP:", clientNode.Identity.String())

	clientNode.DHT.WAN.EnableMitigation()
	clientNode.DHT.WAN.SetProvideRegionSize(20)

	var attackDetectedLog []bool
	var klLog []float64

	var provideCount int
	var netSize int32
	for {
		provideCount++
		log.Info.Printf("%d) Providing for the %d time:", provideCount, provideCount)

		netSize, err = clientNode.DHT.WAN.NsEstimator.NetworkSize()
		if err != nil {
			err = clientNode.DHT.WAN.GatherNetsizeData(ctx)
			if err != nil {
				log.Error.Println("Context error:", ctx.Err())
				err := clientNode.Close()
				if err != nil {
					return
				}

				continue
			}

			netSize, err = clientNode.DHT.WAN.NsEstimator.NetworkSize()
			if err != nil {
				log.Info.Println("Network size estimator error:", err)
				err := clientNode.Close()
				if err != nil {
					return
				}

				continue
			}

			log.Error.Println("Network size estimation:", netSize)
		} else {
			log.Info.Println("Obtained cached network size estimation:", netSize)
		}

		// From their code, but simplified
		clientNode.DHT.WAN.Detector.UpdateIdealDistFromNetsize(int(netSize))
		closest, err := cpl.GetCurrentClosestAsString(ctx, cid, clientNode, 1*time.Minute)
		if err != nil {
			panic(err)
		}
		inCpl := cpl.CountInCpl(cid, closest)
		attackDetected := clientNode.DHT.WAN.Detector.DetectFromCounts(inCpl)
		kl := clientNode.DHT.WAN.Detector.ComputeKLFromCounts(inCpl)

		log.Info.Printf("Attack detected: %t (KL: %f)", attackDetected, kl)
		attackDetectedLog = append(attackDetectedLog, attackDetected)
		klLog = append(klLog, kl)

		// In our tests we don't need to have the content locally, but we could modify the code to do it so.
		// cid, err := ipfs.Unixfs().Add(ctx, files.NewReaderFile(file), OnlyProvideLocally)
		// if err != nil {
		// 	return
		// }
		// log.Info.Println("CID published locally:", cid.String())

		if attackDetected {
			log.Info.Println("Executing region-based queries mitigation...")

			mitigationStart := time.Now()
			log.Info.Println("Forcing the mitigation to publish...")
			err, peers, _, _ := clientNode.DHT.WAN.ProvideWithReturn(ctx, cid, true)
			log.Info.Printf("Published PR to peers %d peers:\n%q\n", len(peers), peers)
			if err != nil {
				log.Error.Println("Error when providing the content using the region-based queries mitigation:")
				panic(err)
			}

			elapsedTime := fmtDuration(time.Since(mitigationStart))
			log.Info.Printf("It took %s to provide! (real duration: %s)\n", elapsedTime, fmtDuration(time.Since(start)))
		} else {
			log.Info.Println("Executing standard publication...")
			err := clientNode.DHT.WAN.Provide(ctx, cid, true)
			if err != nil {
				log.Error.Println("Error when providing the content using a standard publication:")
				panic(err)
			}
		}

		log.Info.Printf("Running until the next interval (%s) or external signal...", config.DefaultReproviderInterval.String())
		shouldStop := runUntilTimeout(clientNode, cancel, 5*time.Minute)
		if shouldStop {
			break
		}
	}

	log.Info.Println("CSV)")
	fmt.Println("id;detected;kl")
	for i := 0; i < provideCount; i++ {
		fmt.Printf("%d;%t;%f\n", i, attackDetectedLog[i], klLog[i])
	}
}

func runUntilTimeout(node *core.IpfsNode, cancel func(), timeout time.Duration) bool {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-c:
		log.Info.Printf("Received signal: %s, shutting down...\n", sig)
		cancel()

		if err := node.Close(); err != nil {
			panic(err)
		}

		return true
	case <-time.After(timeout):
		log.Info.Printf("Timeout of %s reached, shutting down...\n", timeout)
		return false
	}
}
