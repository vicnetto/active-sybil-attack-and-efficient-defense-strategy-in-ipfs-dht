package main

import (
	"flag"
	"fmt"
	"github.com/vicnetto/active-sybil-attack/utils/k-closest-cpl/cpl"
	"os"
	"strings"
	"time"

	gocid "github.com/ipfs/go-cid"
)

var k = 20

type FlagConfig struct {
	cid *string
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flagConfig.cid = flag.String("cid", "", "Goal eclipsed CID")
	flag.Parse()

	var Usage = func() {
		_, err := fmt.Fprintf(os.Stderr, "Usage of ./k-closest-clp [flags]:\n")
		if err != nil {
			return
		}

		flag.PrintDefaults()
	}

	missingFlag := false

	if len(*flagConfig.cid) == 0 {
		fmt.Println("error: flag cid missing.")
		missingFlag = true
	}

	if missingFlag {
		Usage()
		os.Exit(1)
	}

	return &flagConfig
}

func main() {
	flagConfig := treatFlags()

	// Start the experiment:
	fmt.Printf("Getting closest peers to %s...\n\n", *flagConfig.cid)

	var closestList []string
	for {
		closest, err := cpl.GetCurrentClosest(*flagConfig.cid, 60*time.Second)
		if closest == "" || err != nil {
			fmt.Println("Retrying get closest peers...")
			continue
		}

		closestList = strings.Split(strings.TrimSpace(closest), "\n")
		break
	}

	decode, err := gocid.Decode(*flagConfig.cid)
	if err != nil {
		fmt.Println(err)
		return
	}

	counts := cpl.CountInCpl(decode, closestList)

	fmt.Println("- Per CPL -")
	fmt.Printf("CPL:   ")

	sum := 0
	for i := 0; i <= cpl.KeySize; i++ {
		if sum == k {
			break
		}

		if counts[i] != 0 {
			fmt.Printf("%3d ", i)
			sum += counts[i]
		}
	}

	fmt.Printf("\nCount: ")
	sum = 0
	for i := 0; i <= cpl.KeySize; i++ {
		if sum == k {
			break
		}

		if counts[i] != 0 {
			fmt.Printf("%3d ", counts[i])
			sum += counts[i]
		}
	}
	fmt.Println()

	return
}
