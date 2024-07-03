package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/multiformats/go-multiaddr"
	"os"
	"os/signal"
	"sybil/friends"
	instanciate "sybil/instantiate"
	"syscall"
	"time"

	gocid "github.com/ipfs/go-cid"
)

type FlagConfig struct {
	isActive *bool

	privateKey     *string
	eclipsedCid    *string
	ip             *string
	port           int
	sybilsFilePath *string
}

func configForSybil(flagConfig FlagConfig) instanciate.PeerConfig {
	peerConfig := instanciate.PeerConfig{}

	peerConfig.EclipsedCid = flagConfig.eclipsedCid
	peerConfig.Port = flagConfig.port
	peerConfig.Ip = flagConfig.ip
	peerConfig.SybilFilePath = flagConfig.sybilsFilePath

	marshaledPublic, err := base64.StdEncoding.DecodeString(*flagConfig.privateKey)
	if err != nil {
		panic(fmt.Errorf("decode error: %s", err))
	}

	unmarshalPrivate, err := crypto.UnmarshalPrivateKey(marshaledPublic)
	if err != nil {
		panic(fmt.Errorf("unmarshal error: %s", err))
	}

	public := unmarshalPrivate.GetPublic()
	peerId, err := peer.IDFromPublicKey(public)
	if err != nil {
		panic(fmt.Errorf("id from public key error: %s", err))
	}

	peerConfig.Identity = config.Identity{PeerID: peerId.String(), PrivKey: *flagConfig.privateKey}

	return peerConfig
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
		fmt.Println("error: one mode should be specified.")
		missingFlag = true
	}

	if len(*flagConfig.privateKey) == 0 {
		fmt.Println("error: flag privateKey missing.")
		missingFlag = true
	}

	if len(*flagConfig.eclipsedCid) == 0 {
		fmt.Println("error: flag cid missing.")
		missingFlag = true
	}

	if len(*flagConfig.ip) == 0 {
		fmt.Println("error: flag ip missing.")
		missingFlag = true
	}

	if flagConfig.port == 0 {
		fmt.Println("error: flag port missing.")
		missingFlag = true
	}

	if missingFlag {
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	return &flagConfig
}

func printTimestamp() {
	fmt.Printf("[%s] ", time.Now().Format(time.RFC3339))
}

func main() {
	flagConfig := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())

	sybilConfig := configForSybil(*flagConfig)

	printTimestamp()
	fmt.Println("Initializing instantiate with following parameters:")
	fmt.Println("	- PeerID:", sybilConfig.Identity.PeerID)
	fmt.Println("	- Port:", sybilConfig.Port)
	fmt.Println("	- Eclipsed CID:", *sybilConfig.EclipsedCid)
	fmt.Println("	- Eclipsed CID:", *sybilConfig.EclipsedCid)
	fmt.Println("	- Active mode:", *flagConfig.isActive, "\n")

	var otherSybils []multiaddr.Multiaddr
	if len(*flagConfig.sybilsFilePath) != 0 {
		printTimestamp()
		otherSybils = friends.ReadAndFormatOtherPeers(*flagConfig.sybilsFilePath, *flagConfig.privateKey)
	}

	nodeApi, node, err := instanciate.SpawnEphemeral(ctx, sybilConfig, otherSybils)
	if err != nil {
		panic(err)
	}

	// Set attack configuration to avoid IP filters and specify the CID to be eclipsed
	dht.SetAttackConfiguration(*flagConfig.eclipsedCid, *flagConfig.isActive)
	dht.SetGroupToBypassDiversityFilter(friends.ExtractGroupFromIp(*flagConfig.ip))
	rcmgr.SetAllowedIpForSubnetLimit(*flagConfig.ip)

	decode, err := gocid.Decode(*flagConfig.eclipsedCid)
	if err != nil {
		fmt.Println("error: invalid CID passed as parameter.")
		fmt.Println(err)
		return
	}

	// Get closest peers to establish the node in the network
	peers, err := node.DHT.WAN.GetClosestPeers(ctx, string(decode.Hash()))
	printTimestamp()
	fmt.Printf("Closest peers to %s: %q\n\n", decode.String(), peers)

	myAddresses, err := nodeApi.Swarm().ListenAddrs(ctx)
	printTimestamp()
	fmt.Println("My addresses:", myAddresses, "\n")

	// Only if has other sybils to connect
	if len(otherSybils) != 0 {
		duration := time.Duration(5*len(otherSybils)) * time.Second
		printTimestamp()
		fmt.Println("Sleeping for", duration, "before connecting to other sybils...", "\n")

		time.Sleep(duration)
		friends.ConnectToOtherSybils(ctx, nodeApi, node, otherSybils)
	}

	run(node, cancel)
}

func run(node *core.IpfsNode, cancel func()) {
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	<-c

	fmt.Printf("\rExiting...\n")

	cancel()

	if err := node.Close(); err != nil {
		panic(err)
	}

	return
}
