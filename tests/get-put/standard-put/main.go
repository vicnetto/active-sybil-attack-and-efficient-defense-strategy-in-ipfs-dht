package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ipfs/boxo/files"
	"github.com/ipfs/kubo/core"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var log = logger.InitializeLogger()

type Flags struct {
	filepath string
}

func help() func() {
	return func() {
		fmt.Println("\nUsage:", os.Args[0], "[flags]:")
		fmt.Println("  -filepath <string> -- Path to the file to be uploaded")
		fmt.Println("  -provide <int> -- Number of provides (default: 11)")
	}
}

func treatFlags() Flags {
	flags := Flags{}

	flag.StringVar(&flags.filepath, "filepath", "", "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if len(flags.filepath) == 0 {
		fmt.Println("error: flag filepath missing.")
		missingFlag = true
	}

	if missingFlag {
		flag.Usage()
		os.Exit(1)
	}

	return flags
}

func provideFile(ctx context.Context, flags Flags) *core.IpfsNode {
	// Open the file to be uploaded
	file, err := os.Open(flags.filepath)
	if err != nil {
		log.Error.Println(err)
		return nil
	}

	clientConfig := ipfspeer.ConfigForNormalClient(0)
	clientIpfs, clientNode, err := ipfspeer.SpawnEphemeral(ctx, clientConfig)
	if err != nil {
		panic(err)
	}
	log.Info.Println("Peer is UP:", clientNode.Identity.String())

	// Get the file
	cid, err := clientIpfs.Unixfs().Add(ctx, files.NewReaderFile(file))
	if err != nil {
		return nil
	}

	log.Info.Println("Providing CID:", cid.RootCid())

	err = <-dht.ProvideFinished
	if err != nil {
		log.Error.Printf("Provide error for %s in peer %s.", cid.String(), clientNode.Identity.String())
	}

	log.Info.Println("Sleeping 30 seconds to wait provide...")
	time.Sleep(30 * time.Second)

	log.Info.Println("Providing finished!")

	return clientNode
}

func main() {
	flags := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for len(dht.ProvideFinished) > 0 {
		<-dht.ProvideFinished
	}

	clientNode := provideFile(ctx, flags)

	log.Info.Println("Running until exited...")
	run(clientNode, cancel)
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
