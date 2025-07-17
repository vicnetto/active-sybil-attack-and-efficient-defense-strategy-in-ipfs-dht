package generate

import (
	"bufio"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
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
		panic(fmt.Errorf("failed to open file: %w", err))
	}
	defer file.Close()

	var peers []PeerInfo
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 3 {
			panic(fmt.Errorf("invalid line format: [privateKey publicKey port]"))
		}

		portInt, err := strconv.Atoi(parts[2])
		if err != nil {
			panic(fmt.Errorf("invalid port: %w", err))
		}

		peers = append(peers, PeerInfo{
			PrivateKey: parts[0],
			PeerID:     parts[1],
			Port:       portInt,
		})
	}
	if err := scanner.Err(); err != nil {
		panic(fmt.Errorf("error reading file: %w", err))
	}

	return peers
}

func ReadAndFormatPeersAsAddrInfo(peers []PeerInfo, ip string) []peer.AddrInfo {
	var peersMultiaddress []peer.AddrInfo

	for _, peerInfo := range peers {
		multiAddress := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ip, peerInfo.Port, peerInfo.PeerID)
		cast := multiaddr.StringCast(multiAddress)

		addrInfo, err := peer.AddrInfoFromP2pAddr(cast)
		if err != nil {
			panic(fmt.Errorf("failed when parsing multiaddress: %s", cast))
		}

		peersMultiaddress = append(peersMultiaddress, *addrInfo)
	}

	return peersMultiaddress
}
