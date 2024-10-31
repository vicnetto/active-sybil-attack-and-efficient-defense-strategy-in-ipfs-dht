package generate

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

var Timeout = 120 * time.Second

const MaxProbabilities = 40

type NodePerCpl struct {
	Cpl      int
	Quantity int
}

type PeerInfo struct {
	PeerID     string
	PrivateKey string `json:",omitempty"`
	Port       int
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

func ReadAndFormatPeers(filePath string) []PeerInfo {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Failed when opening file")
		panic(err)
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			panic(err)
		}
	}(file)

	var peerInfo []PeerInfo
	scanner := bufio.NewScanner(file)
	fmt.Println("Identities:")

	for i := 0; scanner.Scan(); i++ {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) != 3 {
			panic(fmt.Errorf("invalid line format. line should have the following format [privateKey publicKey port]"))
		}

		privateKey := parts[0]
		peerId := parts[1]
		port := parts[2]

		portInt, err := strconv.Atoi(port)
		if err != nil {
			panic(fmt.Errorf("invalid port. line should have the following format [privateKey publicKey port(int)]"))
		}

		info := PeerInfo{PeerID: peerId, PrivateKey: privateKey, Port: portInt}
		peerInfo = append(peerInfo, info)

		fmt.Println(i, info)
	}
	fmt.Println()

	if err = scanner.Err(); err != nil {
		fmt.Println("Failed when reading file")
		panic(err)
	}

	return peerInfo
}
