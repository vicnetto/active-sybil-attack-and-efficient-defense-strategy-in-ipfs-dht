package main

import (
	"context"
	"flag"
	"fmt"
	gocid "github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/core"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"github.com/vicnetto/active-sybil-attack/utils/xor-distance/mitigation"
	"k-closest-to-file/interact"
	_ "net/http/pprof"
	"os"
	"time"
)

var log = logger.InitializeLogger()

type Flags struct {
	quantity int
}

func help() func() {
	return func() {
		fmt.Println("\nUsage:", os.Args[0], "[flags]:")
		fmt.Println("  -quantity <string>  -- Quantity of DHT lookups to perform")
	}
}

func treatFlags() Flags {
	flags := Flags{}

	flag.IntVar(&flags.quantity, "quantity", 0, "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if flags.quantity == 0 {
		log.Info.Println("error: flag quantity missing.")
		missingFlag = true
	}

	if missingFlag {
		flag.Usage()
		os.Exit(1)
	}

	return flags
}

func main() {
	flags := treatFlags()

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	clientConfig := ipfspeer.ConfigForNormalClient(0)
	_, clientNode, err := ipfspeer.SpawnEphemeral(ctx, clientConfig)
	defer func(clientNode *core.IpfsNode) {
		err := clientNode.Close()
		if err != nil {

		}
	}(clientNode)
	if err != nil {
		panic(err)
	}
	log.Info.Println("PID is up:", clientNode.Identity.String())

	log.Info.Printf("Sleeping for 10 seconds...")
	time.Sleep(10 * time.Second)

	for currentTest := 0; currentTest < flags.quantity; currentTest++ {
		var cidDecode gocid.Cid
		var peers []peer.ID

		log.Info.Printf("%d) Performing random lookups to verify the average distances calculated:", currentTest+1)
		cidDecode, peers = mitigation.PerformRandomLookupReturningQueriedPeers(ctx, clientNode)

		err := interact.StoreDHTLookupToFile(cidDecode, peers, "output")
		if err != nil {
			log.Error.Println("Error while storing DHT lookup to file:", err)
			return
		}
	}
	clientNode.Close()

	// lookups, _ := getRandomDHTLookups(100, "db")

	// var index int
	// for cid, ids := range lookups {
	// 	log.Info.Printf("%d) CID: %s - contains %d peers", index+1, cid.String(), len(ids))
	// 	// for i, id := range ids {
	// 	// 	log.Info.Printf("  %3d. %s", i+1, id.String())
	// 	// }
	// 	index++
	// }

	log.Info.Printf("Finished!")
}
