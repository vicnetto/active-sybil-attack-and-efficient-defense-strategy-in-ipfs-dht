package main

import (
	"context"
	"flag"
	gocid "github.com/ipfs/go-cid"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var log = logger.InitializeLogger()
var dbPath = "../../../db"

type FlagConfig struct {
	cid             string
	estimationTests int
	estimationPeers int
	perfectingPeers int
	perfectingTests int
}

func help() func() {
	return func() {
		log.Info.Println("Usage:", os.Args[0], "[flags]:")
		log.Info.Println("  -cid <string>           -- CID to provide")
		log.Info.Println("  -estimationTests <int>  -- Quantity of tests to a random CID for the first distance estimation")
		log.Info.Println("  -estimationPeers <int>  -- Max peers to contact for obtaining the distance average")
		log.Info.Println("  -perfectingTests <int>  -- Quantity of tests to a random CID for perfecting the distance estimation")
		log.Info.Println("  -perfectingPeers <int>  -- Max peers to contact for perfecting the distance average in a second moment")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flag.StringVar(&flagConfig.cid, "cid", "", "")
	flag.IntVar(&flagConfig.estimationTests, "estimationTests", 0, "")
	flag.IntVar(&flagConfig.estimationPeers, "estimationPeers", 0, "")
	flag.IntVar(&flagConfig.perfectingTests, "perfectingTests", 0, "")
	flag.IntVar(&flagConfig.perfectingPeers, "perfectingPeers", 0, "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if len(flagConfig.cid) == 0 {
		log.Error.Println("error: flag cid missing.")
		missingFlag = true
	}

	if flagConfig.estimationTests == 0 {
		log.Error.Println("error: flag estimationTests missing.")
		missingFlag = true
	}

	if flagConfig.estimationPeers == 0 {
		log.Error.Println("error: flag estimationPeers missing.")
		missingFlag = true
	}

	if flagConfig.perfectingTests == 0 {
		log.Error.Println("error: flag perfectingTests missing.")
		missingFlag = true
	}

	if flagConfig.perfectingPeers == 0 {
		log.Error.Println("error: flag perfectingPeers missing.")
		missingFlag = true
	}

	if missingFlag {
		log.Error.Println()
		flag.Usage()
		os.Exit(1)
	}

	return &flagConfig
}

func main() {
	flagConfig := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())

	peerConfig := ipfspeer.ConfigForRandomNode(0)

	_, clientNode, err := ipfspeer.SpawnEphemeral(ctx, peerConfig)
	if err != nil {
		panic(err)
	}
	log.Info.Println("PID is UP:", clientNode.Identity.String())

	log.Info.Println("Sleep for 10 seconds before starting...")
	time.Sleep(10 * time.Second)

	cid, err := gocid.Parse(flagConfig.cid)
	if err != nil {
		panic("Invalid CID. Please insert a valid one.")
	}
	err = clientNode.DHT.WAN.SRProvide(ctx, cid, true)
	if err != nil {
		log.Error.Println("Error while executing the SR-DHT-Store")
		panic(err)

	}
	// dk, providedTo, err := clientNode.DHT.WAN.GetClosestPeersAndProvideIfWithinDk(ctx, string(cid.Hash()))
	// log.Info.Println("Closests:", dk)
	// log.Info.Println("ProvidedTo:", providedTo)

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	<-c

	log.Info.Printf("Exiting...\n")

	cancel()

	if err := clientNode.Close(); err != nil {
		panic(err)
	}
}
