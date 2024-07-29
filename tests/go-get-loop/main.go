package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-errors/errors"
	"github.com/ipfs/boxo/path"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"log"
	"os"
	"time"

	gocid "github.com/ipfs/go-cid"
)

type Flags struct {
	cid                   *string
	numberOfTests         int
	recurrent             int
	numberOfVerifications int
}

func help() func() {
	return func() {
		fmt.Println("\nUsage:", os.Args[0], "[flags]:")
		fmt.Println("  -verifications <int> -- Number of verifications (default: 1)")
		fmt.Println("  -tests <int> -- Number of tests per verification (default: 20)")
		fmt.Println("  -recurrent <int> -- Minutes between each verification (default: 30 minutes)")
	}
}

func treatFlags() *Flags {
	flags := Flags{}
	flags.cid = flag.String("cid", "", "CID to be obtained")
	flag.IntVar(&flags.numberOfVerifications, "verifications", 1, "Number of verifications")
	flag.IntVar(&flags.numberOfTests, "tests", 20, "Number of tests per verification")
	flag.IntVar(&flags.recurrent, "recurrent", 30, "Minutes between each verification")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if len(*flags.cid) == 0 {
		fmt.Println("error: flag cid missing.")
		missingFlag = true
	}

	if missingFlag {
		flag.Usage()
		os.Exit(1)
	}

	return &flags
}

func savePrProviders(destination map[string]int, origin map[string]int) map[string]int {
	for key, value := range origin {
		destination[key] += value
	}

	return destination
}

func printStats(fileObtained int, eclipsePrProviders map[string]int, filePrProviders map[string]int, numberOfTests int) {
	fmt.Printf("** Eclipsed: %d (%.2f%%)\n", numberOfTests-fileObtained,
		float32(numberOfTests-fileObtained)/float32(numberOfTests)*100)

	for key, value := range eclipsePrProviders {
		fmt.Printf("*** [%d: %s]\n", value, key)
	}

	fmt.Printf("** Obtained: %d (%.2f%%)\n", fileObtained,
		float32(fileObtained)/float32(numberOfTests)*100)

	for key, value := range filePrProviders {
		fmt.Printf("*** [%d: %s]\n", value, key)
	}
}

func main() {
	flags := treatFlags()

	location, _ := time.LoadLocation("Europe/Paris")

	globalFilePrProviders := map[string]int{}
	globalEclipsePrProviders := map[string]int{}
	var globalFileObtained int

	// Create the context
	decodedCid, err := gocid.Decode(*flags.cid)
	if err != nil {
		fmt.Println(err)
		return
	}

	pathCid := path.FromCid(decodedCid)

	for verification := 1; verification <= flags.numberOfVerifications; verification++ {
		var fileObtained int
		filePrProviders := map[string]int{}
		eclipsePrProviders := map[string]int{}

		for i := 1; i <= flags.numberOfTests; i++ {
			fmt.Printf("\n%s\n", time.Now().In(location).Format("02-01-2006-15:04:05-CEST"))
			ctx, cancel := context.WithCancel(context.Background())

			fmt.Printf("%d) Instantiating node\n", i)
			clientConfig := ipfspeer.ConfigForNormalClient(65000)
			clientIPFS, clientNode, err := ipfspeer.SpawnEphemeral(ctx, clientConfig)
			if err != nil {
				panic(err)
			}

			ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)

			// Add file to our local storage
			_, err = clientIPFS.Unixfs().Get(ctx, pathCid)

			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					if len(dht.GetPRProvider()) == 0 {
						fmt.Println("No provider found!")
					}

					fmt.Println("File eclipsed!")
					eclipsePrProviders[dht.GetPRProvider()]++
				} else {
					fmt.Println("Error getting the file:", err)
					continue
				}
			} else {
				fileObtained++
				fmt.Println("File obtained!")

				filePrProviders[dht.GetPRProvider()]++
			}

			err = clientNode.Close()
			if err != nil {
				panic(err)
			}

			fmt.Println("Sleeping 5 seconds after stopping node...")
			time.Sleep(5 * time.Second)
			dht.ResetPRProvider()

			cancel()
		}

		fmt.Printf("\n%s\n", time.Now().In(location).Format("02-01-2006-15:04:05-CEST"))
		fmt.Printf("* Stats %d (total of %d tests) >\n", verification, flags.numberOfTests)
		printStats(fileObtained, eclipsePrProviders, filePrProviders, flags.numberOfTests)

		fmt.Printf("\n%s\n", time.Now().In(location).Format("02-01-2006-15:04:05-CEST"))
		fmt.Printf("* Global stats of %d verifications (total of %d tests) >\n", verification, flags.numberOfTests*verification)
		globalFileObtained += fileObtained
		globalEclipsePrProviders = savePrProviders(globalEclipsePrProviders, eclipsePrProviders)
		globalFilePrProviders = savePrProviders(globalFilePrProviders, filePrProviders)
		printStats(globalFileObtained, globalEclipsePrProviders, globalFilePrProviders, flags.numberOfTests*verification)

		if verification+1 > flags.numberOfVerifications {
			return
		}

		fmt.Println()
		log.Println("Sleeping for", flags.recurrent, "minutes before continuing...")
		time.Sleep(time.Duration(flags.recurrent) * time.Minute)
	}
}
