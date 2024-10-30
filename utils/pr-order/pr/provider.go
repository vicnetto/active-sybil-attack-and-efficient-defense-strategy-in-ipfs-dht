package pr

import (
	"context"
	gocid "github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/core"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-base32"
	"github.com/vicnetto/active-sybil-attack/logger"
	"strings"
	"time"
)

type SmallestProvider struct {
	Pid    peer.ID
	Base32 string
}

func GetProvidersFromPeer(ctx context.Context, logger logger.Logger, clientNode *core.IpfsNode,
	node peer.ID, targetCid gocid.Cid, timeLimitForPeerResponse time.Duration) ([]*peer.AddrInfo, error) {
	ctxTimeout, ctxTimeoutCancel := context.WithTimeout(ctx, timeLimitForPeerResponse)
	defer ctxTimeoutCancel()

	if _, err := clientNode.DHT.WAN.FindPeer(ctxTimeout, node); err != nil {
		logger.Error.Println("Failed to connect to peer", node.String())
		logger.Error.Println(err)

		return nil, err
	}

	providers, _, err := clientNode.DHT.WAN.ProtoMessenger.GetProviders(ctxTimeout, node, targetCid.Hash())
	if err != nil {
		logger.Error.Println("Failed trying to ask peer", node.String(), "for providers and closest")
		logger.Error.Println(err)

		return nil, err
	}

	return providers, err
}

func FindSmallestBase32(currentSmallest SmallestProvider, providers []*peer.AddrInfo) SmallestProvider {
	if len(currentSmallest.Pid) == 0 && len(providers) != 0 {
		currentSmallest.Pid = providers[0].ID
		currentSmallest.Base32 = base32.RawStdEncoding.EncodeToString([]byte(providers[0].ID))
	}

	if len(providers) > 0 {
		for _, provider := range providers {
			keyPeerEncoded := base32.RawStdEncoding.EncodeToString([]byte(provider.ID))

			if strings.Compare(keyPeerEncoded, currentSmallest.Base32) < 0 {
				currentSmallest.Pid = provider.ID
				currentSmallest.Base32 = keyPeerEncoded
			}
		}
	}

	return currentSmallest
}

func ProvideCidToPeer(ctx context.Context, logger logger.Logger, clientNode *core.IpfsNode, pid peer.ID,
	cid gocid.Cid, timeLimitForPeerResponse time.Duration) error {
	ctxTimeout, ctxTimeoutCancel := context.WithTimeout(ctx, timeLimitForPeerResponse)
	defer ctxTimeoutCancel()

	if _, err := clientNode.DHT.WAN.FindPeer(ctxTimeout, pid); err != nil {
		logger.Error.Println("Failed to connect to peer", pid.String())
		logger.Error.Println(err)

		return err
	}

	myPid := clientNode.Identity
	myAddresses := clientNode.PeerHost.Addrs()
	err := clientNode.DHT.WAN.ProtoMessenger.PutProviderAddrs(ctxTimeout, pid, cid.Hash(), peer.AddrInfo{
		ID:    myPid,
		Addrs: myAddresses,
	})
	if err != nil {
		logger.Error.Printf("Failed to provide %s to peer %s.\n", cid.String(), pid.String())
		logger.Error.Println(err)

		return err
	}

	return err
}
