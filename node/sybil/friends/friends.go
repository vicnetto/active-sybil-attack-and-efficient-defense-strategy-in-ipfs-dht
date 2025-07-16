package friends

import (
	"bufio"
	"context"
	"fmt"
	"github.com/ipfs/kubo/core"
	coreiface "github.com/ipfs/kubo/core/coreiface"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/vicnetto/active-sybil-attack/logger"
	"net"
	"os"
	"strings"
	"time"
)

var sleepBetweenConnections = 2 * time.Second

var log = logger.InitializeLogger()

func ExtractGroupFromIp(address string) string {
	ip := net.ParseIP(address)
	ipParts := strings.Split(address, ".")
	if ip == nil || len(ipParts) != 4 {
		panic(fmt.Errorf("invalid ip address: %s", address))
	}

	return fmt.Sprintf("%s.%s.0.0", ipParts[0], ipParts[1])
}

func ExtractUniqueIPv4(address []multiaddr.Multiaddr) []string {
	ipMap := make(map[string]bool)

	for _, currentMultiAddress := range address {
		current := currentMultiAddress.String()

		if strings.HasPrefix(current, "/ip4/") {
			ip := current[len("/ip4/"):]
			endIndex := strings.Index(ip, "/")
			if endIndex != -1 {
				ip = ip[:endIndex]
			}

			ipParts := strings.Split(ip, ".")
			if len(ipParts) == 4 {
				ip = fmt.Sprintf("%s.%s.0.0", ipParts[0], ipParts[1])
			}

			ipMap[ip] = true
		}
	}

	// Converter as chaves do mapa em uma slice
	var uniqueIPs []string
	for ip := range ipMap {
		uniqueIPs = append(uniqueIPs, ip)
	}

	return uniqueIPs
}

func ReadOtherPeersAsPeerInfo(filePath string, myPrivateKey string, ip string) []peer.AddrInfo {
	file, err := os.Open(filePath)
	if err != nil {
		log.Error.Println("Failed when opening file in the filepath:", filePath)
		panic(err)
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			panic(err)
		}
	}(file)

	var otherSybilsAddrInfo []peer.AddrInfo
	scanner := bufio.NewScanner(file)
	// log.Info.Println("Other sybils:")

	for i := 0; scanner.Scan(); i++ {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) != 3 {
			panic(fmt.Errorf("invalid line format. line should have the following format [privateKey publicKey port]"))
		}

		privateKey := parts[0]
		peerId := parts[1]
		port := parts[2]

		if privateKey == myPrivateKey {
			continue
		}

		multiAddress := fmt.Sprintf("/ip4/%s/tcp/%s/p2p/%s", ip, port, peerId)
		cast := multiaddr.StringCast(multiAddress)

		addrInfo, err := peer.AddrInfoFromP2pAddr(cast)
		if err != nil {
			log.Error.Println("Failed when parsing MultiAddress:", cast)
			panic(err)
		}
		// addrInfo.Addrs = nil

		otherSybilsAddrInfo = append(otherSybilsAddrInfo, *addrInfo)

		// log.Info.Println(i, addrInfo)
	}
	// fmt.Println()

	if err = scanner.Err(); err != nil {
		log.Error.Println("Failed when reading file")
		panic(err)
	}

	return otherSybilsAddrInfo
}

func ConnectToOtherSybils(ctx context.Context, ipfsApi coreiface.CoreAPI, ipfsNode *core.IpfsNode, otherSybils []multiaddr.Multiaddr) {
	fmt.Println("Connecting to other sybils...")

	addrInfos, err := peer.AddrInfosFromP2pAddrs(otherSybils...)
	if err != nil {
		fmt.Println("Error getting addrInfos from P2pAddresses: ", err)
		return
	}

	// tries := 0
	for i := 0; i < len(addrInfos); i++ {
		peerId := addrInfos[i].ID
		fmt.Printf("Connecting to peer %d: %s\n", i+1, peerId)

		// err = ipfsApi.Swarm().Connect(ctx, addrInfos[i])
		// if err == nil {
		// 	fmt.Println("Connected SWARM:", peerId)
		// } else {
		// 	fmt.Println("Failed SWARM:", peerId)
		// 	fmt.Println(err)
		// }

		// Add to the routing table
		_, err = ipfsNode.DHT.WAN.RoutingTable().TryAddPeer(peerId, true, false)
		if err == nil {
			fmt.Println("Connected to RT:", peerId)
		} else {
			fmt.Println(err)
			i--
		}
		fmt.Println()

		time.Sleep(sleepBetweenConnections)
	}
}
