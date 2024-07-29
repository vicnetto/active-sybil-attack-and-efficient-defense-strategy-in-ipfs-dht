package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ipfs/boxo/files"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreiface/options"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type FlagConfig struct {
	filename *string
}

func DontProvide(options *options.UnixfsAddSettings) error {
	options.OnlyLocal = true

	return nil
}

func fmtDuration(duration time.Duration) string {
	m := int(duration.Minutes()) % 60
	s := int(duration.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d", m, s)
}

func help() func() {
	return func() {
		fmt.Println("Usage:", os.Args[0], "[flags]:")
		fmt.Println("  -filename <string>    -- Path of the file to be published.")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flagConfig.filename = flag.String("filename", "", "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if len(*flagConfig.filename) == 0 {
		fmt.Println("error: flag filename missing.")
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

	// Open the file to be uploaded
	file, err := os.Open(*flagConfig.filename)
	if err != nil {
		log.Println(err)
		return
	}

	start := time.Now()
	peerConfig := ipfspeer.ConfigForNormalClient(8080)

	for {
		ctx, cancel := context.WithCancel(context.Background())

		ipfs, node, err := ipfspeer.SpawnEphemeral(ctx, peerConfig)
		if err != nil {
			panic(err)
		}

		_, netSizeErr := node.DHT.WAN.NsEstimator.NetworkSize()
		if netSizeErr != nil {
			err = node.DHT.WAN.GatherNetSizeData(ctx)
			if err != nil {
				log.Println("Context error:", ctx.Err())
				err := node.Close()
				if err != nil {
					return
				}

				continue
			}

			_, netSizeErr = node.DHT.WAN.NsEstimator.NetworkSize()
			if netSizeErr != nil {
				log.Println("NetSize error:", netSizeErr)
				err := node.Close()
				if err != nil {
					return
				}

				continue
			}
		}

		cid, err := ipfs.Unixfs().Add(ctx, files.NewReaderFile(file), DontProvide)
		if err != nil {
			return
		}

		log.Println("\nCID published in local:", cid.String())

		log.Println("\nPublishing with the mitigation...")
		node.DHT.WAN.EnableMitigation()
		node.DHT.WAN.SetProvideRegionSize(20)
		err, peers, _, _ := node.DHT.WAN.ProvideWithReturn(ctx, cid.RootCid(), true)
		log.Printf("Published PR to peers %d peers:\n%q\n", len(peers), peers)
		if err != nil {
			return
		}

		elapsedTime := fmtDuration(time.Since(start))
		log.Println("It took", elapsedTime, "to upload!")
		log.Println("Running until canceled...")

		file.Close()
		run(node, cancel)

		break
	}
}

func run(node *core.IpfsNode, cancel func()) {
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	<-c

	log.Printf("\rExiting...\n")

	cancel()

	if err := node.Close(); err != nil {
		panic(err)
	}

	return
}
