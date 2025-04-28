package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"github.com/vicnetto/active-sybil-attack/utils/k-closest-cpl/cpl"
	"os"
	"time"

	gocid "github.com/ipfs/go-cid"
)

var k = 20

var log = logger.InitializeLogger()

type FlagConfig struct {
	cid string
}

func help() func() {
	return func() {
		fmt.Println("Usage:", os.Args[0], "[flags]:")
		fmt.Println("    -cid -- Tested CID")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flag.StringVar(&flagConfig.cid, "cid", "", "")

	missingFlag := false
	flag.Usage = help()
	flag.Parse()

	if len(flagConfig.cid) == 0 {
		log.Info.Println("error: flag cid missing.")
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

	var cid, err = gocid.Parse(flagConfig.cid)
	if err != nil {
		log.Error.Println("Invalid cid.")
		return
	}

	clientConfig := ipfspeer.ConfigForNormalClient(0)
	_, clientNode, err := ipfspeer.SpawnEphemeral(ctx, clientConfig)
	if err != nil {
		panic(err)
	}

	log.Info.Println("PID is up:", clientNode.Identity.String())

	log.Info.Printf("Sleeping for 10 seconds...")
	time.Sleep(10 * time.Second)

	closest, err := cpl.GetCurrentClosestAsString(ctx, cid, clientNode, time.Second*30)
	counts := cpl.CountInCpl(cid, closest)

	log.Info.Println("Results)")
	log.Info.Println("- Per CPL -")

	var cplString string
	sum := 0
	for i := 0; i <= cpl.KeySize; i++ {
		if sum == k {
			break
		}

		if counts[i] != 0 {
			cplString += fmt.Sprintf("%3d ", i)
			sum += counts[i]
		}
	}

	var countString string
	sum = 0
	for i := 0; i <= cpl.KeySize; i++ {
		if sum == k {
			break
		}

		if counts[i] != 0 {
			countString += fmt.Sprintf("%3d ", counts[i])
			sum += counts[i]
		}
	}

	log.Info.Println("CPL  :", cplString)
	log.Info.Println("Count:", countString)

	cancel()
	return
}
