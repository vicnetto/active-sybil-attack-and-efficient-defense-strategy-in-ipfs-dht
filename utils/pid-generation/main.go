package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/vicnetto/active-sybil-attack/utils/pid-generation/generate"
	"os"
	"runtime"
	"strconv"
)

func writePeersToOutputFile(pidGenerateConfig generate.PidGenerateConfig, peerId []string, privateKey []string) {
	file, err := os.Create(pidGenerateConfig.OutFile)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	// Create a bufio.Writer to efficiently write to the file.
	writer := bufio.NewWriter(file)

	// Write lines into the file.
	for i := 0; i < len(privateKey); i++ {
		_, err := writer.WriteString(fmt.Sprintf("%s %s %d\n", privateKey[i], peerId[i], pidGenerateConfig.FirstPort+i))

		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
	}

	// Flush the bufio.Writer to ensure all data is written to the file.
	err = writer.Flush()
	if err != nil {
		fmt.Println("Error flushing writer:", err)
		return
	}

	fmt.Println("File ", pidGenerateConfig.OutFile, " created!")
}

func help() func() {
	return func() {
		fmt.Println("Usage: ./generate-PID [mode] [flags]:")
		fmt.Println(" A mode must be specified:")
		fmt.Println("	-byBase32            -- Generate PIDs closer than the specified peer in Base32")
		fmt.Println("	-byInterval          -- Generate PIDs within a specific interval from the CID")
		fmt.Println("	-byClosest           -- Generate PIDs closer than the cpl instantiate to the CID")
		fmt.Println("	-byCpl               -- Generate PIDs with a specific common prefix length (CPL) from the CID")
		fmt.Println(" Global flags:")
		fmt.Println("	-firstPort <int>     -- Initial port for generated sybils (default: 10000)")
		fmt.Println("	-outFile <string>    -- Output file name (default: sybils-out)")
		fmt.Println("	-useAllCpus <bool>   -- Use all CPUs for the calculation (default: true)")
		fmt.Println(" Flags for -byBase32 mode:")
		fmt.Println("	-peer <string>       -- Reference peer")
		fmt.Println("	-quantity <int>      -- Number of peer IDs")
		fmt.Println(" Flags for -byClosest mode:")
		fmt.Println("	-cid <string>        -- Referenced CID")
		fmt.Println("	-quantity <int>      -- Number of peer IDs")
		fmt.Println(" Flags for -byInterval mode:")
		fmt.Println("	-cid <string>        -- Referenced CID")
		fmt.Println("	-quantity <int>      -- Number of peer IDs")
		fmt.Println("	-firstPeer <string>  -- First peer in the interval")
		fmt.Println("	-secondPeer <string> -- Second peer in the interval")
		fmt.Println(" Flags for -byCpl mode:")
		fmt.Println("	-cid <string>        -- Referenced CID")
		fmt.Println("	-<int> <int>         -- Specify the number of nodes to be generated for each CPL. Multiple CPLs can be specified.")
		fmt.Println("	                         Example: -10 5 -11 7")
		fmt.Println("	                                  |     *-----> CPL: 11, quantity: 7")
		fmt.Println("	                                  *-----------> CPL: 10, quantity: 5")
	}
}

func treatFlags() *generate.PidGenerateConfig {
	flagConfig := generate.PidGenerateConfig{}

	byInterval := flag.Bool("byInterval", false, "")
	byClosest := flag.Bool("byClosest", false, "")
	byCpl := flag.Bool("byCpl", false, "")
	byBase32 := flag.Bool("byBase32", false, "")

	var quantityInCpl [generate.MaxProbabilities]int
	for i := 0; i < generate.MaxProbabilities; i++ {
		flag.IntVar(&quantityInCpl[i], strconv.Itoa(i), 0, "")
	}
	quantity := flag.Int("quantity", 0, "")
	cpl := flag.Int("cpl", 0, "")
	firstPort := flag.Int("firstPort", 10000, "")
	useAllCpus := flag.Bool("useAllCpus", true, "")
	cid := flag.String("cid", "", "")
	outFile := flag.String("outFile", "sybils-out", "")
	firstPeer := flag.String("firstPeer", "", "")
	secondPeer := flag.String("secondPeer", "", "")
	referencePeer := flag.String("peer", "", "")

	flag.Usage = help()

	flag.Parse()

	flagConfig.ByInterval = *byInterval
	flagConfig.ByClosest = *byClosest
	flagConfig.ByCpl = *byCpl
	flagConfig.ByBase32 = *byBase32
	flagConfig.Quantity = *quantity
	flagConfig.Cpl = *cpl
	flagConfig.FirstPort = *firstPort
	flagConfig.UseAllCpus = *useAllCpus
	flagConfig.Cid = *cid
	flagConfig.OutFile = *outFile
	flagConfig.FirstPeer = *firstPeer
	flagConfig.SecondPeer = *secondPeer
	flagConfig.ReferencePeer = *referencePeer

	if !flagConfig.ByInterval && !flagConfig.ByClosest && !flagConfig.ByCpl && !flagConfig.ByBase32 {
		fmt.Println("error: mode missing.")
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	missingFlag := false

	if flagConfig.ByCpl {
		for i := 0; i < generate.MaxProbabilities; i++ {
			if quantityInCpl[i] > 0 {
				nodeInCpl := generate.NodePerCpl{Cpl: i, Quantity: quantityInCpl[i]}
				flagConfig.NodesPerCpl = append(flagConfig.NodesPerCpl, nodeInCpl)
			}
		}

		if len(flagConfig.NodesPerCpl) == 0 {
			fmt.Println("error: flag -<cpl> <quantity> missing.")
			missingFlag = true
		}
	}

	if flagConfig.ByInterval {
		if len(flagConfig.FirstPeer) == 0 {
			fmt.Println("error: flag firstPeer missing.")
			missingFlag = true
		}

		if len(flagConfig.SecondPeer) == 0 {
			fmt.Println("error: flag secondPeer missing.")
			missingFlag = true
		}
		if flagConfig.Quantity == 0 {
			fmt.Println("error: flag quantity missing.")
			missingFlag = true
		}
	}

	if flagConfig.ByClosest || flagConfig.ByInterval || flagConfig.ByBase32 {
		if flagConfig.Quantity == 0 {
			fmt.Println("error: flag quantity missing.")
			missingFlag = true
		}
	}

	if flagConfig.ByBase32 {
		if len(flagConfig.ReferencePeer) == 0 {
			fmt.Println("error: flag peer missing.")
			missingFlag = true
		}
	} else {
		if len(flagConfig.Cid) == 0 {
			fmt.Println("error: flag cid missing.")
			missingFlag = true
		}
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

	var numberCpu int
	if flagConfig.UseAllCpus {
		numberCpu = runtime.NumCPU()
		fmt.Printf("Using all %d CPUs for generating the peers...\n\n", numberCpu)
	} else {
		numberCpu = 1
		fmt.Printf("Using only one CPU for generating the peers...\n\n")
	}
	runtime.GOMAXPROCS(numberCpu)

	var peerId []string
	var privateKey []string

	var closestList []string
	if !flagConfig.ByBase32 {
		fmt.Printf("Getting closest peers to %s...\n\n", flagConfig.Cid)
		closestList = generate.GetClosestPeersFromCidAsList(*flagConfig)
	}

	if flagConfig.ByCpl {
		var cplPeerId []string
		var cplPrivateKey []string

		for _, nodesInCpl := range flagConfig.NodesPerCpl {
			flagConfig.Cpl = nodesInCpl.Cpl
			flagConfig.Quantity = nodesInCpl.Quantity

			cplPeerId, cplPrivateKey, _ = generate.GeneratePeers(*flagConfig, numberCpu, closestList)
			peerId = append(peerId, cplPeerId...)
			privateKey = append(privateKey, cplPrivateKey...)
		}
	} else {
		peerId, privateKey, _ = generate.GeneratePeers(*flagConfig, numberCpu, closestList)
	}

	writePeersToOutputFile(*flagConfig, peerId, privateKey)

	return
}
