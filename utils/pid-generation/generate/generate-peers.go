package generate

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ipfs/go-cid"
	kspace "github.com/libp2p/go-libp2p-kbucket/keyspace"
	mh "github.com/multiformats/go-multihash"
	"log"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"time"

	gocid "github.com/ipfs/go-cid"
)

var IpfsPath = "ipfs"

type Interval struct {
	farthest *big.Int
	closest  *big.Int
}

func GetClosestPeersFromCidAsList(pidGenerateConfig PidGenerateConfig) []string {
	var closestList []string

	for {
		closest, err := GetClosestPeersFromCID(*pidGenerateConfig.Cid, Timeout)
		if closest == "" || err != nil {
			fmt.Println("Retrying get cpl peers...")
			continue
		}

		closestList = strings.Split(strings.TrimSpace(closest), "\n")
		break
	}

	return closestList
}

func GetClosestPeersFromCID(CID string, timeout time.Duration) (string, error) {
	// Requires IPFS daemon to be running
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, IpfsPath, "dht", "query", CID)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("GetClosestPeersFromCID() with CID: %s failed", CID)
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		os.Exit(1)
	}

	return out.String(), err
}

func GetClosestToCIDFromPeersList(cid cid.Cid, peers []string) (string, *big.Int) {
	var closestPeer string
	var minDistance *big.Int

	cidByte, _ := mh.FromB58String(cid.String())
	cidKey := kspace.XORKeySpace.Key(cidByte)

	for i, peer := range peers {
		peerByte, _ := mh.FromB58String(peer)
		peerKey := kspace.XORKeySpace.Key(peerByte)
		distance := peerKey.Distance(cidKey)

		if i == 0 {
			closestPeer = peer
			minDistance = distance
			continue
		}

		if distance.Cmp(minDistance) == -1 {
			closestPeer = peer
			minDistance = distance
		}
	}

	return closestPeer, minDistance
}

func generateKeysInMultipleCpus(pidGenerateConfig PidGenerateConfig, numberCpu int, interval Interval,
	targetCidKey kspace.Key) ([]string, []string, error) {

	var peerId []string
	var privateKey []string

	wgCount := &WaitGroupCount{}
	quit := make(chan bool)
	result := make(chan KeyInfo)
	availableCpu := numberCpu
	var foundRoutine int32

	start := time.Now()
	for i := 0; i < pidGenerateConfig.Quantity; i++ {
		for availableCpu != 0 {
			availableCpu--
			wgCount.Add(1)
			go GenerateValidKey(pidGenerateConfig, interval, targetCidKey, quit, result, wgCount, &foundRoutine)
		}

		var keyInfo KeyInfo
		for {
			select {
			case keyInfo = <-result:
				availableCpu++

				fmt.Printf("Found in: %d tries\n\tDistance: %d\n\tSybil PeerID: %s\n\tSybil Private Key: %s\n",
					keyInfo.tries, keyInfo.peerDistance, keyInfo.peerId, keyInfo.marshalPrivateKey)
				fmt.Printf("%d/%d peers found in %s!\n\n", i+1, pidGenerateConfig.Quantity, time.Since(start))

				peerId = append(peerId, keyInfo.peerId)
				privateKey = append(privateKey, keyInfo.marshalPrivateKey)

				break
			}
			break
		}
	}

	// Stop all currently opened GoRoutines
	for wgCount.GetCount() != 0 {
		quit <- true
	}

	wgCount.Wait()
	close(result)
	close(quit)

	fmt.Println("All GoRoutines stopped! Finished!")

	since := time.Since(start)
	fmt.Println("It took", since, "to generate peers.")
	fmt.Println()

	return peerId, privateKey, nil
}

func GeneratePeers(pidGenerateConfig PidGenerateConfig, numberCpu int, closestList []string) ([]string, []string, error) {
	decode, err := gocid.Decode(*pidGenerateConfig.Cid)
	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}

	targetCIDByte, _ := mh.FromB58String(*pidGenerateConfig.Cid)
	targetCIDKey := kspace.XORKeySpace.Key(targetCIDByte)

	var interval Interval

	if *pidGenerateConfig.ByCpl {
		interval = ByCplConfiguration(pidGenerateConfig, decode, targetCIDKey, closestList)
	}

	if *pidGenerateConfig.ByInterval {
		interval = ByIntervalConfiguration(pidGenerateConfig, targetCIDKey)
	}

	if *pidGenerateConfig.ByClosest {
		interval = ByClosestConfiguration(pidGenerateConfig, targetCIDKey, closestList)
	}

	var peerId []string
	var privateKey []string
	peerId, privateKey, _ = generateKeysInMultipleCpus(pidGenerateConfig, numberCpu, interval, targetCIDKey)

	return peerId, privateKey, nil
}
