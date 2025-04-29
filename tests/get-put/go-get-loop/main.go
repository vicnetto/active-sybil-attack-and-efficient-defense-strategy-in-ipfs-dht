package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ipfs/boxo/path"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	_ "net/http/pprof"
	"os"
	"runtime"
	"time"

	gocid "github.com/ipfs/go-cid"
)

var log = logger.InitializeLogger()

type FlagConfig struct {
	cid                   string
	providerPeerId        string
	numberOfTests         int
	recurrent             int
	numberOfVerifications int
}

func help() func() {
	return func() {
		fmt.Println("\nUsage:", os.Args[0], "[flags]:")
		fmt.Println("  -cid <string> -- CID to be tested")
		fmt.Println("  -providerPeerId <string> -- CID to be tested")
		fmt.Println("  -verifications <int> -- Number of verifications (default: 1)")
		fmt.Println("  -tests <int> -- Number of tests per verification (default: 20)")
		fmt.Println("  -recurrent <int> -- Minutes between each verification (default: 30 minutes)")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}
	flag.StringVar(&flagConfig.cid, "cid", "", "CID to be obtained")
	flag.StringVar(&flagConfig.providerPeerId, "providerPeerId", "", "")
	flag.IntVar(&flagConfig.numberOfVerifications, "verifications", 1, "Number of verifications")
	flag.IntVar(&flagConfig.numberOfTests, "tests", 20, "Number of tests per verification")
	flag.IntVar(&flagConfig.recurrent, "recurrent", 30, "Minutes between each verification")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if len(flagConfig.cid) == 0 {
		fmt.Println("error: flag cid missing.")
		missingFlag = true
	}

	if len(flagConfig.providerPeerId) == 0 {
		fmt.Println("error: flag providerPeerId missing.")
		missingFlag = true
	}

	if missingFlag {
		flag.Usage()
		os.Exit(1)
	}

	return &flagConfig
}

func savePrProviders(destination map[string]int, origin map[string]int) map[string]int {
	for key, value := range origin {
		destination[key] += value
	}

	return destination
}

func printStats(fileObtained int, eclipsePrProviders map[string]int, filePrProviders map[string]int, numberOfTests int) {
	log.Info.Printf("** Eclipsed: %d (%.2f%%)\n", numberOfTests-fileObtained,
		float32(numberOfTests-fileObtained)/float32(numberOfTests)*100)

	for key, value := range eclipsePrProviders {
		log.Info.Printf("*** [%d: %s]\n", value, key)
	}

	log.Info.Printf("** Obtained: %d (%.2f%%)\n", fileObtained,
		float32(fileObtained)/float32(numberOfTests)*100)

	for key, value := range filePrProviders {
		log.Info.Printf("*** [%d: %s]\n", value, key)
	}
}

func verifyIfFileIsEclipsed(cidPath path.ImmutablePath, fileObtained *int,
	eclipsePrProviders map[string]int, filePrProviders map[string]int) {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	clientConfig := ipfspeer.ConfigForNormalClient(65000)
	clientIPFS, clientNode, err := ipfspeer.SpawnEphemeral(ctx, clientConfig)
	if err != nil {
		panic(err)
	}
	defer clientNode.Close()

	log.Info.Println("PID is up:", clientNode.Identity.String())

	ctxTimeout, ctxTimeoutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ctxTimeoutCancel()

	// Get the file
	_, err = clientIPFS.Unixfs().Get(ctxTimeout, cidPath)
	found, prProvider := dht.GetLookupInformation()

	if found {
		*fileObtained++
		log.Info.Println("File obtained!")

		filePrProviders[prProvider]++
	} else {
		if len(prProvider) == 0 {
			log.Info.Println("No provider found!")
			prProvider = "No provider found"
		}

		log.Info.Println("File eclipsed!")
		eclipsePrProviders[prProvider]++
	}
}

func main() {
	// go func() {
	// 	fmt.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	flags := treatFlags()

	globalFilePrProviders := map[string]int{}
	globalEclipsePrProviders := map[string]int{}
	var globalFileObtained int

	decodedCid, err := gocid.Decode(flags.cid)
	if err != nil {
		log.Error.Println(err)
		return
	}

	pathCid := path.FromCid(decodedCid)
	dht.SetRealProvider(flags.providerPeerId)

	for verification := 1; verification <= flags.numberOfVerifications; verification++ {
		var fileObtained int
		filePrProviders := map[string]int{}
		eclipsePrProviders := map[string]int{}

		for i := 1; i <= flags.numberOfTests; i++ {
			verifyIfFileIsEclipsed(pathCid, &fileObtained, eclipsePrProviders, filePrProviders)

			runtime.GC()
			log.Info.Println("Sleeping 5 seconds after stopping node...")
			dht.ResetPRProvider()
		}

		log.Info.Printf("* Verification %d (total of %d tests) >\n", verification, flags.numberOfTests)
		printStats(fileObtained, eclipsePrProviders, filePrProviders, flags.numberOfTests)

		log.Info.Printf("* Global stats of %d verifications (total of %d tests) >\n", verification, flags.numberOfTests*verification)
		globalFileObtained += fileObtained
		globalEclipsePrProviders = savePrProviders(globalEclipsePrProviders, eclipsePrProviders)
		globalFilePrProviders = savePrProviders(globalFilePrProviders, filePrProviders)
		printStats(globalFileObtained, globalEclipsePrProviders, globalFilePrProviders, flags.numberOfTests*verification)

		if verification+1 > flags.numberOfVerifications {
			return
		}

		runtime.GC()
		fmt.Println()
		log.Info.Println("Sleeping for", flags.recurrent, "minutes before continuing...")
		time.Sleep(time.Duration(flags.recurrent) * time.Minute)
	}
}
