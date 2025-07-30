package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/core"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"
)

var log = logger.InitializeLogger()

type Flags struct {
	cid          string
	providerPid  string
	maxProviders int
}

func help() func() {
	return func() {
		fmt.Println("\nUsage:", os.Args[0], "[flags]:")
		fmt.Println("  -cid <string>       -- CID to do the DHT lookup.")
		fmt.Println("  -provider <string>  -- Real provider of the content.")
		fmt.Println("  -maxProviders <int> -- Amount of records to stop the DHT lookup.")
	}
}

func treatFlags() *Flags {
	flags := Flags{}

	flag.StringVar(&flags.cid, "cid", "", "")
	flag.StringVar(&flags.providerPid, "provider", "", "")
	flag.IntVar(&flags.maxProviders, "maxProviders", 0, "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if len(flags.cid) == 0 {
		fmt.Println("error: flag cid missing.")
		missingFlag = true
	}

	if len(flags.providerPid) == 0 {
		fmt.Println("error: flag provider missing.")
		missingFlag = true
	}

	if flags.maxProviders == 0 {
		fmt.Println("error: flag maxProviders missing.")
		missingFlag = true
	}

	if missingFlag {
		flag.Usage()
		os.Exit(1)
	}

	return &flags
}

func lookupForCid(flags Flags) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientConfig := ipfspeer.ConfigForNormalClient(65000)
	_, clientNode, err := ipfspeer.SpawnEphemeral(ctx, clientConfig)
	defer func(clientNode *core.IpfsNode) {
		err := clientNode.Close()
		if err != nil {

		}
	}(clientNode)
	log.Info.Println("PID is up:", clientNode.Identity.String())

	decodedCid, err := cid.Decode(flags.cid)
	if err != nil {
		panic(err)
	}

	log.Info.Println("Sleeping for 5 seconds...")
	time.Sleep(5 * time.Second)

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 10*time.Second)
	defer timeoutCancel()

	log.Info.Printf("Find providers to %s)", flags.cid)
	var providers []string
	for p := range clientNode.DHT.WAN.FindProvidersAsync(timeoutCtx, decodedCid, 0) {
		log.Info.Println("Found provider:", p)
		providers = append(providers, p.ID.String())
	}

	if slices.Contains(providers, flags.providerPid) {
		log.Info.Println("Provider obtained!")
	} else {
		log.Info.Println("Provider eclipsed!")
	}
}

func main() {
	flags := treatFlags()

	lookupForCid(*flags)

	log.Info.Println("Sleeping 5 seconds after stopping node...")
	time.Sleep(5 * time.Second)
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
