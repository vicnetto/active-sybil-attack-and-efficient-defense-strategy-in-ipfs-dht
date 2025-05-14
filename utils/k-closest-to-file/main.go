package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ipfs/kubo/core"
	srutils "github.com/libp2p/go-libp2p-kad-dht/sr/utils"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"github.com/vicnetto/active-sybil-attack/utils/k-closest-to-file/interact"
	_ "net/http/pprof"
	"os"
	"time"
)

var log = logger.InitializeLogger()

type Flags struct {
	quantity   int
	port       int
	privateKey string
}

func help() func() {
	return func() {
		fmt.Println("\nUsage:", os.Args[0], "[flags]:")
		fmt.Println("  -quantity <string>  -- Quantity of DHT lookups to perform")
		fmt.Println("  -port <int>          -- Port of the IPFS node. (default: any valid port)")
		fmt.Println("  -privateKey <string> -- Private key of the IPFS node. (default: random node)")
	}
}

func treatFlags() Flags {
	flagConfig := Flags{}

	flag.IntVar(&flagConfig.quantity, "quantity", 0, "")
	flag.IntVar(&flagConfig.port, "port", 0, "")
	flag.StringVar(&flagConfig.privateKey, "privateKey", "", "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if flagConfig.quantity == 0 {
		log.Info.Println("error: flag quantity missing.")
		missingFlag = true
	}

	if missingFlag {
		flag.Usage()
		os.Exit(1)
	}

	return flagConfig
}

func main() {
	flagConfig := treatFlags()

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	var peerConfig ipfspeer.Config
	if len(flagConfig.privateKey) != 0 {
		peerConfig = ipfspeer.ConfigForSpecificNode(flagConfig.port, flagConfig.privateKey)
	} else {
		peerConfig = ipfspeer.ConfigForRandomNode(flagConfig.port)
	}

	_, clientNode, err := ipfspeer.SpawnEphemeral(ctx, peerConfig)
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

	for currentTest := 0; currentTest < flagConfig.quantity; currentTest++ {
		log.Info.Printf("%d) Performing random lookups to verify the average distances calculated:", currentTest+1)
		cidDecode, allQueriedPeers := srutils.PerformRandomLookupReturningAllQueriedPeers(ctx, clientNode)

		err := interact.StoreDHTLookupToFile(cidDecode, allQueriedPeers, "output")
		if err != nil {
			log.Error.Println("Error while storing DHT lookup to file:", err)
			return
		}
	}

	// Obtain from DB:
	// lookups, err := interact.GetRandomDHTLookups(100, "../../db")
	// if err != nil {
	// 	panic(err)
	// }

	// Obtain distance from each of the peers in the DB:
	// for cid, peers := range lookups {
	// 	targetCIDByte, _ := mh.FromB58String(cid.String())
	// 	targetCIDKey := kspace.XORKeySpace.Key(targetCIDByte)

	// 	aMultiHash, _ := mh.FromB58String(peers[20-1].String())
	// 	aPeerKey := kspace.XORKeySpace.Key(aMultiHash)

	// 	aDistance := aPeerKey.Distance(targetCIDKey)
	// 	fmt.Println(aDistance)
	// }

	// Print peers in the DB:
	// var index int
	//  for cid, ids := range lookups {
	//  log.Info.Printf("%d) CID: %s - contains %d peers", index+1, cid.String(), len(ids))
	//  log.Info.Printf("CID) %s", cid.String())
	//  for i, id := range peers {
	//  	if i == 20 {
	//   		break
	//   	}
	//   	log.Info.Printf("  %3d. %s", i+1, id.String())
	//  }
	//  index++
	// }

	log.Info.Printf("Finished!")
}
