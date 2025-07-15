package main

import (
	"context"
	"flag"
	"fmt"
	gocid "github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/core"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	pidgenerate "github.com/vicnetto/active-sybil-attack/utils/pid-generation/generate"
	"os"
	"os/signal"
	"syscall"
)

var log = logger.InitializeLogger()

type FlagConfig struct {
	privateKey         string
	otherNodesFilepath string
	targetCid          string
	ip                 string
	port               int
}

func help() func() {
	return func() {
		fmt.Println("Usage:", os.Args[0], "[flags]:")
		fmt.Println("    -privateKey <string>         -- Private key.")
		fmt.Println("    -otherNodesFilepath <string> -- Other nodes information (format: pkey pid port) (optional).")
		fmt.Println("    -targetCid <string>          -- From which the other nodes will be recommended (optional).")
		fmt.Println("    -port <int>                  -- Node port. (default: any available port)")
		fmt.Println("    -ip <string>                 -- Node IP address. (default: any ip)")
	}
}

func treatFlags() FlagConfig {
	flagConfig := FlagConfig{}

	flag.StringVar(&flagConfig.privateKey, "privateKey", "", "")
	flag.StringVar(&flagConfig.ip, "ip", "0.0.0.0", "")
	flag.StringVar(&flagConfig.otherNodesFilepath, "otherNodesFilepath", "", "")
	flag.StringVar(&flagConfig.targetCid, "targetCid", "", "")
	flag.IntVar(&flagConfig.port, "port", 0, "")

	flag.Usage = help()

	flag.Parse()

	missingFlag := false

	if len(flagConfig.privateKey) == 0 {
		log.Info.Println("error: flag privateKey missing.")
		missingFlag = true
	}

	if missingFlag {
		log.Info.Println()
		flag.Usage()
		os.Exit(1)
	}

	return flagConfig
}

func main() {
	flagConfig := treatFlags()

	ctx, cancel := context.WithCancel(context.Background())

	sybilConfig := ipfspeer.ConfigForSybil(&flagConfig.ip, flagConfig.port, flagConfig.privateKey)

	var otherNodes []peer.AddrInfo
	var targetCid gocid.Cid
	if len(flagConfig.otherNodesFilepath) != 0 {
		var err error
		log.Info.Println(flagConfig.targetCid)
		targetCid, err = gocid.Decode(flagConfig.targetCid)
		if err != nil {
			log.Error.Println("Invalid CID: ", err)
			panic(err)
		}

		nodesFromFile := pidgenerate.ReadAndFormatPeers(flagConfig.otherNodesFilepath)

		for _, node := range nodesFromFile {
			multiAddress := multiaddr.StringCast(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", flagConfig.ip, node.Port,
				node.PeerID))
			addr, err := peer.AddrInfoFromP2pAddr(multiAddress)
			if err != nil {
				log.Error.Println("When parsing multiaddress:", err)
				panic(err)
			}

			otherNodes = append(otherNodes, *addr)
		}
	}

	// Set parameters to recommend other nodes when asking about the content:
	dht.OtherNodes = otherNodes
	dht.TargetCID = targetCid

	ipfs, node, err := ipfspeer.SpawnEphemeral(ctx, sybilConfig)
	if err != nil {
		panic(err)
	}

	log.Info.Println("Peer is up:", node.Identity.String())

	decode, _ := multihash.Cast([]byte(node.Identity))
	log.Info.Println("Getting closest nodes to itself...")
	peers, err := node.DHT.WAN.GetClosestPeers(ctx, string(decode))
	log.Info.Printf("Closest peers: %q\n", peers)

	myAddresses, err := ipfs.Swarm().ListenAddrs(ctx)
	log.Info.Println("My addresses:", myAddresses)

	log.Info.Println("Running until exited...")

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
