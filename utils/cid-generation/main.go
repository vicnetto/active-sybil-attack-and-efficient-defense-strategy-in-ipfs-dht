package main

import (
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	kspace "github.com/libp2p/go-libp2p-kbucket/keyspace"
	"github.com/libp2p/go-libp2p/core/peer"
	mh "github.com/multiformats/go-multihash"
	"github.com/vicnetto/active-sybil-attack/logger"
	"os"
)

var log = logger.InitializeLogger()

type FlagConfig struct {
	cid string
	cpl int
}

func help() func() {
	return func() {
		log.Info.Println("Usage:", os.Args[0], "[flags]:")
		log.Info.Println("  -cid <string>  -- Reference CID")
		log.Info.Println("  -cpl <int>     -- CPL with the CID")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flag.StringVar(&flagConfig.cid, "cid", "", "")
	flag.IntVar(&flagConfig.cpl, "cpl", 0, "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if len(flagConfig.cid) == 0 {
		log.Error.Println("error: flag cid missing.")
		missingFlag = true
	}

	if flagConfig.cpl == 0 {
		log.Error.Println("error: flag cpl missing.")
		missingFlag = true
	}

	if missingFlag {
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	return &flagConfig
}

var maxCplForRefresh = uint(15)

// From Kubo
func GenRandPeerID(referenceCid kbucket.ID, targetCpl uint) (peer.ID, error) {
	if targetCpl > maxCplForRefresh {
		return "", fmt.Errorf("cannot generate peer ID for Cpl greater than %d", maxCplForRefresh)
	}

	localPrefix := binary.BigEndian.Uint16(referenceCid)

	// For host with ID `L`, an ID `K` belongs to a bucket with ID `B` ONLY IF CommonPrefixLen(L,K) is EXACTLY B.
	// Hence, to achieve a targetPrefix `T`, we must toggle the (T+1)th bit in L & then copy (T+1) bits from L
	// to our randomly generated prefix.
	toggledLocalPrefix := localPrefix ^ (uint16(0x8000) >> targetCpl)
	randPrefix, err := randUint16()
	if err != nil {
		return "", err
	}

	// Combine the toggled local prefix and the random bits at the correct offset
	// such that ONLY the first `targetCpl` bits match the local ID.
	mask := (^uint16(0)) << (16 - (targetCpl + 1))
	targetPrefix := (toggledLocalPrefix & mask) | (randPrefix & ^mask)

	// Convert to a known peer ID.
	key := keyPrefixMap[targetPrefix]
	id := [32 + 2]byte{mh.SHA2_256, 32}
	binary.BigEndian.PutUint32(id[2:], key)
	return peer.ID(id[:]), nil
}

func randUint16() (uint16, error) {
	// Read a random prefix.
	var prefixBytes [2]byte
	_, err := rand.Read(prefixBytes[:])
	return binary.BigEndian.Uint16(prefixBytes[:]), err
}

func main() {
	flagConfig := treatFlags()

	// cidByte := kbucket.ConvertPeerID(peer.ID(flagConfig.cid))
	peerMultiHash, _ := mh.FromB58String(flagConfig.cid)
	peerKey := kspace.XORKeySpace.Key(peerMultiHash)
	id, err := GenRandPeerID(peerKey.Bytes, uint(flagConfig.cpl))
	if err != nil {
		log.Error.Printf("Error generating random CID with %d CPL: %s", flagConfig.cpl, err)
		panic(err)
	}

	log.Info.Println("Reference CID:", flagConfig.cid)
	log.Info.Printf("%d CPL CID: %s\n", flagConfig.cpl, id.String())
}
