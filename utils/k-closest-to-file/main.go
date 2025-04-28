package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ipfs/kubo/core"
	"github.com/libp2p/go-libp2p-kad-dht/qpeerset"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"github.com/vicnetto/active-sybil-attack/utils/k-closest-to-file/interact"
	"github.com/vicnetto/active-sybil-attack/utils/xor-distance/mitigation"
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
		log.Info.Printf("%d) Performing random lookups to verify the average distances calculated:", currentTest+1)
		cidDecode, lookupResult := mitigation.PerformRandomLookupReturningQueriedPeersWithFullInformation(ctx, clientNode)
		peers := lookupResult.AllPeersContacted.GetClosestInStates(qpeerset.PeerHeard, qpeerset.PeerWaiting, qpeerset.PeerQueried)

		err := interact.StoreDHTLookupToFile(cidDecode, peers, "output")
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
