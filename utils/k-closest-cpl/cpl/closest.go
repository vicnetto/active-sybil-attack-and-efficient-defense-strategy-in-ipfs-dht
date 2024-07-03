package cpl

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"log"
	"os/exec"
	"time"

	kb "github.com/libp2p/go-libp2p-kbucket"
)

var IpfsPath = "ipfs"
var KeySize = 256

func GetCurrentClosest(CID string, timeout time.Duration) (string, error) {
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, IpfsPath, "dht", "query", CID)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("getCurrentClosest() with CID: %s failed", CID)
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
	}

	return out.String(), err
}

func CountInCpl(cid cid.Cid, closestPeersToCid []string) []int {
	counts := make([]int, KeySize)

	// Convert CID to hash and after to bytes
	cidHash := cid.Hash()
	targetCid := kb.ConvertKey(string(cidHash))

	// Compare the two values and count how many appearances per CPL
	for _, pid := range closestPeersToCid {
		pidDecoded, _ := peer.Decode(pid)
		pidBytes := kb.ConvertKey(string(pidDecoded))
		prefixLen := kb.CommonPrefixLen(targetCid, pidBytes)
		counts[prefixLen]++
	}

	return counts
}
