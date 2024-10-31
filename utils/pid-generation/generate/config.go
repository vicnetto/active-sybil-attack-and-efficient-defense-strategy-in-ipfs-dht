package generate

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

var Timeout = 120 * time.Second

const MaxProbabilities = 40

type NodePerCpl struct {
	Cpl      int
	Quantity int
}

type PidGenerateConfig struct {
	Quantity   int
	Cid        string
	FirstPort  int
	OutFile    string
	UseAllCpus bool

	ByInterval bool
	FirstPeer  string
	SecondPeer string

	ByClosest   bool
	ByCpl       bool
	Cpl         int
	NodesPerCpl []NodePerCpl

	ByBase32      bool
	ReferencePeer string
}

func WritePeersToOutputFile(pidGenerateConfig PidGenerateConfig, peerId []string, privateKey []string) {
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
