package generate

import (
	"crypto/rand"
	"fmt"
	kspace "github.com/libp2p/go-libp2p-kbucket/keyspace"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	mh "github.com/multiformats/go-multihash"
	"log"
	"math/big"
	"os"
	"sync"
	"sync/atomic"
)

type KeyInfo struct {
	tries             int
	peerId            string
	peerKey           kspace.Key
	peerDistance      *big.Int
	marshalPrivateKey string
}

type WaitGroupCount struct {
	sync.WaitGroup
	count int64
}

func (wg *WaitGroupCount) Add(delta int) {
	atomic.AddInt64(&wg.count, int64(delta))
	wg.WaitGroup.Add(delta)
}

func (wg *WaitGroupCount) Done() {
	atomic.AddInt64(&wg.count, -1)
	wg.WaitGroup.Done()
}

func (wg *WaitGroupCount) GetCount() int {
	return int(atomic.LoadInt64(&wg.count))
}

func generateNewKey(targetCIDKey kspace.Key) (KeyInfo, error) {
	privateKey, publicKey, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	peerId, err := peer.IDFromPublicKey(publicKey)
	if err != nil {
		log.Println(err)
		return KeyInfo{}, err
	}

	peerMultiHash, _ := mh.FromB58String(peerId.String())
	peerKey := kspace.XORKeySpace.Key(peerMultiHash)
	peerDistance := peerKey.Distance(targetCIDKey)

	marshalPrivateKey, err := crypto.MarshalPrivateKey(privateKey)
	if err != nil {
		fmt.Printf("Error when marshalling private key")
		return KeyInfo{}, err
	}

	encodedMarshalPrivateKey := crypto.ConfigEncodeKey(marshalPrivateKey)

	return KeyInfo{peerId: peerId.String(), peerKey: peerKey, peerDistance: peerDistance,
		marshalPrivateKey: encodedMarshalPrivateKey}, nil
}

func GenerateValidKey(pidGenerateConfig PidGenerateConfig, interval Interval, targetCidKey kspace.Key, quit <-chan bool,
	result chan<- KeyInfo, wg *WaitGroupCount, found *int32) {
	tries := 0

	for {
		select {
		case <-quit:
			wg.Done()
			return

		default:
			tries = tries + 1
			key, err := generateNewKey(targetCidKey)
			if err != nil {
				continue
			}

			good := false
			if pidGenerateConfig.ByCpl {
				good = IsValidAccordingToCPLRules(pidGenerateConfig.Cpl, interval, key, targetCidKey)
			}

			if pidGenerateConfig.ByInterval {
				good = IsValidAccordingToIntervalRules(interval, key.peerDistance)
			}

			if pidGenerateConfig.ByClosest {
				good = IsValidAccordingToClosestRules(interval, key)
			}

			if good {
				if int(*found) != pidGenerateConfig.Quantity {
					atomic.AddInt32(found, 1)

					key.tries = tries
					result <- key
					wg.Done()

					return
				}
			}
		}
	}
}
