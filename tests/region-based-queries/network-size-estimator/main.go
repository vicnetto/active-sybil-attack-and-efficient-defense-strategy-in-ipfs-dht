package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"os"
	"time"
)

var log = logger.InitializeLogger()

type FlagConfig struct {
	privateKey string
	port       int
	filename   string
	tests      int
}

type NetworkSizeResult struct {
	currentTime string
	elapsedTime string
	networkSize int
}

func appendOutputToFile(filename string, line string) {
	// Open the filename in append mode, create it if it doesn't exist
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error.Printf("Error opening filename: %v\n", err)
		return
	}
	defer file.Close()

	// Write the line to the filename
	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(line)
	if err != nil {
		log.Error.Printf("Error writing to filename: %v\n", err)
		return
	}

	// Flush the buffered writer to ensure the line is written to the filename
	err = writer.Flush()
	if err != nil {
		log.Error.Printf("Error flushing writer: %v\n", err)
		return
	}
}

func fmtDuration(duration time.Duration) string {
	m := int(duration.Minutes()) % 60
	s := int(duration.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d", m, s)
}

func help() func() {
	return func() {
		fmt.Println("Usage:", os.Args[0], "[flags]:")
		fmt.Println("  -tests <int>         -- Number of tests.")
		fmt.Println("  -privateKey <string> -- Private key of the IPFS node. (default: random node)")
		fmt.Println("  -port <int>          -- Port of the IPFS node. (default: any valid port)")
		fmt.Println("  -out <string>        -- File to write the output.")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flag.StringVar(&flagConfig.filename, "filename", "", "")
	flag.StringVar(&flagConfig.privateKey, "privateKey", "", "")
	flag.IntVar(&flagConfig.tests, "tests", 0, "")
	flag.IntVar(&flagConfig.port, "port", 0, "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if len(flagConfig.filename) == 0 {
		fmt.Println("error: flag filename missing.")
		missingFlag = true
	}

	if flagConfig.tests == 0 {
		fmt.Println("error: flag tests missing.")
		missingFlag = true
	}

	if missingFlag {
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	return &flagConfig
}

func main() {
	flagConfig := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())

	var peerConfig ipfspeer.Config
	if len(flagConfig.privateKey) != 0 {
		peerConfig = ipfspeer.ConfigForSpecificNode(flagConfig.port, flagConfig.privateKey)
	} else {
		peerConfig = ipfspeer.ConfigForRandomNode(flagConfig.port)
	}

	var networkSizeResults []NetworkSizeResult

	for i := 0; i < flagConfig.tests; i++ {
		log.Info.Printf("%d) Test %d", i, i)

		_, clientNode, err := ipfspeer.SpawnEphemeral(ctx, peerConfig)
		if err != nil {
			panic(err)
		}

		start := time.Now()
		netSize, err := clientNode.DHT.WAN.NsEstimator.NetworkSize()
		if err != nil {
			err = clientNode.DHT.WAN.GatherNetsizeData(ctx)
			if err != nil {
				log.Error.Printf("  %s.. retrying!", err)
				err = clientNode.Close()
				i--

				continue
			}

			netSize, err = clientNode.DHT.WAN.NsEstimator.NetworkSize()
			if err != nil {
				log.Error.Printf("  %s.. retrying!", err)
				err = clientNode.Close()
				i--

				continue
			}
		}

		networkSizeResult := NetworkSizeResult{
			currentTime: time.Now().Format(time.RFC3339),
			elapsedTime: fmtDuration(time.Since(start)),
			networkSize: int(netSize),
		}
		networkSizeResults = append(networkSizeResults, networkSizeResult)

		log.Info.Println(" Result)")
		log.Info.Printf("  CurrentTime: %s\n", networkSizeResult.currentTime)
		log.Info.Printf("  ElapsedTime: %s\n", networkSizeResult.elapsedTime)
		log.Info.Printf("  NetworkSize: %d\n", networkSizeResult.networkSize)

		line := fmt.Sprintf("%s;%s;%d\n", networkSizeResult.currentTime, networkSizeResult.elapsedTime, networkSizeResult.networkSize)
		appendOutputToFile(flagConfig.filename, line)

		err = clientNode.Close()
		log.Info.Println()
	}

	log.Info.Println("Results)")
	log.Info.Println("id;currentTime;elapsedTime;networkSize")
	for i, networkSizeResult := range networkSizeResults {
		log.Info.Printf("%d;%s;%s;%d\n", i+1, networkSizeResult.currentTime, networkSizeResult.elapsedTime, networkSizeResult.networkSize)
	}

	cancel()
}
