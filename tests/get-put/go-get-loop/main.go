package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/gofrs/flock"
	"github.com/ipfs/kubo/core"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"time"

	gocid "github.com/ipfs/go-cid"
)

var log = logger.InitializeLogger()

const logsDir = "logs"
const lockDir = "lock"

type Flags struct {
	cidFilepath   string
	privateKey    string
	providerPid   string
	interval      time.Duration
	port          int
	verifications int
	iteration     int
}

type CidStatus struct {
	filePrProviders     map[peer.ID]int
	eclipsePrProviders  map[peer.ID]int
	eclipseCount        int
	lastTestWasEclipsed bool
	lastTest            time.Time
}

func help() func() {
	return func() {
		fmt.Println("\nUsage:", os.Args[0], "[flags]:")
		fmt.Println("  -privateKey <string>  -- Private key of the test node")
		fmt.Println("  -port <int>           -- Port to run the test node")
		fmt.Println("  -cidFilepath <string> -- CIDs to be tested for every test")
		fmt.Println("  -providerPid <string> -- Peer ID of the provider")
		fmt.Println("  -interval <int>       -- Minutes between each verification (in minutes) (default: 30)")
		fmt.Println("  -verifications <int>  -- Number of verifications (default: 1)")
		fmt.Println("  -iteration <int>      -- In case of long tests, inform the iteration for the log file (default: 1)")
	}
}

func treatFlags() Flags {
	flags := Flags{}
	var interval int
	flag.StringVar(&flags.privateKey, "privateKey", "", "")
	flag.StringVar(&flags.cidFilepath, "cidFilepath", "", "")
	flag.StringVar(&flags.providerPid, "providerPid", "", "")
	flag.IntVar(&interval, "interval", 30, "")
	flag.IntVar(&flags.verifications, "verifications", 1, "")
	flag.IntVar(&flags.port, "port", 10000, "")
	flag.IntVar(&flags.iteration, "iteration", 1, "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if len(flags.cidFilepath) == 0 {
		log.Info.Println("error: flag cid missing.")
		missingFlag = true
	}

	if len(flags.providerPid) == 0 {
		log.Info.Println("error: flag providerPid missing.")
		missingFlag = true
	}

	flags.interval = time.Duration(interval) * time.Minute

	if missingFlag {
		flag.Usage()
		os.Exit(1)
	}

	return flags
}

func initializeMapOfCidStatus(cidList []gocid.Cid) map[gocid.Cid]CidStatus {
	var allCidStatus = map[gocid.Cid]CidStatus{}
	for _, cid := range cidList {
		status := CidStatus{}
		status.filePrProviders = make(map[peer.ID]int)
		status.eclipsePrProviders = make(map[peer.ID]int)
		allCidStatus[cid] = status
	}

	return allCidStatus
}

func printStats(status CidStatus, numberOfTests int) {
	log.Info.Printf("** Eclipsed: %d (%.2f%%)\n", status.eclipseCount,
		float32(status.eclipseCount)/float32(numberOfTests)*100)

	for key, value := range status.eclipsePrProviders {
		log.Info.Printf("*** [%d: %s]\n", value, key)
	}

	log.Info.Printf("** Obtained: %d (%.2f%%)\n", numberOfTests-status.eclipseCount,
		float32(numberOfTests-status.eclipseCount)/float32(numberOfTests)*100)

	for key, value := range status.filePrProviders {
		log.Info.Printf("*** [%d: %s]\n", value, key)
	}
}

func verifyIfFileIsEclipsed(ctx context.Context, clientNode *core.IpfsNode, cid gocid.Cid, status CidStatus, providerPid peer.ID) CidStatus {
	for {
		ctxTimeout, ctxTimeoutCancel := context.WithTimeout(ctx, 10*time.Second)
		// Get the providers in a loop until the context has finished. By setting to 0, we search for all the records
		// as possible.
		for range clientNode.DHT.WAN.FindProvidersAsync(ctxTimeout, cid, 0) {
		}

		_, ok := dht.RecordReceivedFrom[providerPid]
		if ok {
			log.Info.Println("File obtained!")

			status.filePrProviders[""]++
			status.lastTestWasEclipsed = false
		} else {
			if len(dht.RecordReceivedFrom) == 0 {
				log.Info.Println("No provider found! Sleeping one minute before continuing...")
				// time.Sleep(1 * time.Minute)
				// ctxTimeoutCancel()
				// continue
			}

			status.eclipseCount++
			status.lastTestWasEclipsed = true
			log.Info.Printf("File eclipsed! Received %d records.", len(dht.RecordReceivedFrom))
			status.eclipsePrProviders[""]++
		}

		status.lastTest = time.Now()

		ctxTimeoutCancel()
		return status
	}
}

func readCidListFromFile(cidFilepath string) []gocid.Cid {
	var testCid []gocid.Cid

	file, err := os.Open(cidFilepath)
	if err != nil {
		fmt.Println("Failed when opening cid file")
		panic(err)
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			log.Error.Println("Error when closing file cid file")
			panic(err)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	for i := 0; scanner.Scan(); i++ {
		cidFromFile := scanner.Text()
		parsedCid, err := gocid.Decode(cidFromFile)
		if err != nil {
			log.Error.Println("Error when decoding CID from file. Verify if the CIDs have the right format.")
			os.Exit(1)
		}

		testCid = append(testCid, parsedCid)
	}

	return testCid
}

func decodeIdentifiers(flags Flags) ([]gocid.Cid, peer.ID, ipfspeer.Config) {
	cidList := readCidListFromFile(flags.cidFilepath)

	providerPid, err := peer.Decode(flags.providerPid)
	if err != nil {
		log.Error.Println("Error when decoding the provider PID.")
		os.Exit(1)
	}

	var clientConfig ipfspeer.Config
	if len(flags.privateKey) != 0 {
		ip := "0.0.0.0"
		clientConfig = ipfspeer.ConfigForSybil(&ip, flags.port, flags.privateKey)
	} else {
		clientConfig = ipfspeer.ConfigForRandomNode(flags.port)
	}

	return cidList, providerPid, clientConfig
}

func logCurrentCidStatus(nodePid peer.ID, cidStatus map[gocid.Cid]CidStatus, flags Flags) error {
	// Ensure the "logs" directory exists
	if err := os.MkdirAll(logsDir, os.ModePerm); err != nil {
		log.Error.Printf("could create the log directory: %s", err)
		return err
	}

	if err := os.MkdirAll(lockDir, os.ModePerm); err != nil {
		log.Error.Printf("could create the lock directory: %s", err)
		return err
	}

	for cid, status := range cidStatus {
		filePath := filepath.Join(logsDir, fmt.Sprintf("%d-%s.log", flags.verifications*int(flags.interval.Minutes()), cid.String()))
		lockPath := filepath.Join(lockDir, fmt.Sprintf("%d-%s.lock", flags.verifications*int(flags.interval.Minutes()), cid.String()))

		// Open the file in append mode, create it if it doesn't exist
		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Error.Printf("could not open or create file %s: %s", filePath, err)
			return err
		}

		var lock *flock.Flock
		for {
			lock = flock.New(lockPath)
			locked, err := lock.TryLock()
			if err != nil {
				log.Error.Printf("could not obtain lock for file %s: %s", lockPath, err)
				return err
			}
			if !locked {
				log.Error.Printf("file %s is currently locked by another process", lockPath)
				continue
			}
			break
		}

		outputString := fmt.Sprintf("%s,%s,%d,%d,%t\n",
			status.lastTest.Format("2006-01-02,15:04:05"),
			nodePid.String(),
			flags.verifications*int(flags.interval.Minutes()),
			flags.iteration,
			status.lastTestWasEclipsed)

		if _, err := file.WriteString(outputString); err != nil {
			err := file.Close()
			if err != nil {
				return err
			}
			log.Error.Printf("could not write to file %s: %s", filePath, err)
			return err
		}

		// Close the file
		if err := file.Close(); err != nil {
			log.Error.Printf("could not close file %s: %s", filePath, err)
			return err
		}

		err = lock.Unlock()
		if err != nil {
			log.Error.Printf("could not unlock file %s: %s", lockPath, err)
			return err
		}
	}

	return nil
}

func main() {
	start := time.Now()
	flags := treatFlags()

	cidList, providerPid, clientConfig := decodeIdentifiers(flags)

	log.Info.Println("CIDs to test:", cidList)
	log.Info.Println("Provider:", providerPid.String())

	cidStatus := initializeMapOfCidStatus(cidList)

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	_, clientNode, err := ipfspeer.SpawnEphemeral(ctx, clientConfig)
	defer clientNode.Close()
	if err != nil {
		panic(err)
	}

	log.Info.Println("Peer is UP:", clientNode.Identity.String())

	testStartTime := start
	for current := 1; current <= flags.verifications; current++ {
		for _, cid := range cidList {
			log.Info.Printf("%s) Verifying CID\n", cid)
			cidStatus[cid] = verifyIfFileIsEclipsed(ctx, clientNode, cid, cidStatus[cid], providerPid)
			dht.RecordReceivedFrom = nil
		}

		log.Info.Printf("* Stats of %d tests >\n", current)
		for cid, status := range cidStatus {
			log.Info.Printf("* Stats for %s)\n", cid.String())
			printStats(status, current)
		}

		// Print everything to a log file
		err := logCurrentCidStatus(clientNode.Identity, cidStatus, flags)
		if err != nil {
			return
		}

		// Some maths just to have exactly X minutes between each test
		sleepTime := flags.interval - time.Now().Sub(testStartTime)
		log.Info.Printf("Sleeping for %s...\n", sleepTime)
		time.Sleep(sleepTime)

		// Starting time for the next test
		testStartTime = time.Now()
	}

	log.Info.Printf("Finished iteration %d!", flags.iteration)
}
