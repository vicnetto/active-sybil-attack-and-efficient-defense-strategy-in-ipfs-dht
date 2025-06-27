package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ipfs/kubo/core"
	"github.com/multiformats/go-multihash"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"os"
	"os/signal"
	"syscall"
)

var log = logger.InitializeLogger()

type FlagConfig struct {
	privateKey *string
	ip         *string
	port       int
}

func help() func() {
	return func() {
		fmt.Println("Usage:", os.Args[0], "[flags]:")
		fmt.Println("    -privateKey <string> -- Private key.")
		fmt.Println("    -port <int>          -- Node port. (default: any available port)")
		fmt.Println("    -ip <string>         -- Node IP address. (default: any ip)")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flagConfig.privateKey = flag.String("privateKey", "", "")
	flagConfig.ip = flag.String("ip", "0.0.0.0", "")
	flag.IntVar(&flagConfig.port, "port", 0, "")

	flag.Usage = help()

	flag.Parse()

	missingFlag := false

	if len(*flagConfig.privateKey) == 0 {
		log.Info.Println("error: flag privateKey missing.")
		missingFlag = true
	}

	if missingFlag {
		log.Info.Println()
		flag.Usage()
		os.Exit(1)
	}

	return &flagConfig
}

func main() {
	flagConfig := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())

	sybilConfig := ipfspeer.ConfigForSybil(flagConfig.ip, flagConfig.port, *flagConfig.privateKey)

	ipfs, node, err := ipfspeer.SpawnEphemeral(ctx, sybilConfig)
	if err != nil {
		panic(err)
	}

	log.Info.Println("Peer is up:", node.Identity.String())

	decode, _ := multihash.Cast([]byte(node.Identity))
	log.Info.Println("Getting closest nodes to itself...")
	peers, err := node.DHT.WAN.GetClosestPeers(ctx, string(decode))
	log.Info.Printf("Closest peers: %q\n", peers)

	myAddresses, err := ipfs.Swarm().ListenAddrs(ctx)
	log.Info.Println("My addresses:", myAddresses)

	log.Info.Println("Running until exited...")

	run(node, cancel)
}

func run(node *core.IpfsNode, cancel func()) {
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	<-c

	log.Info.Printf("Exiting...\n")

	cancel()

	if err := node.Close(); err != nil {
		panic(err)
	}

	return
}
