package instantiate

import (
	"context"
	"fmt"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	coreiface "github.com/ipfs/kubo/core/coreiface"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var DefaultIpfsPort = "4001"

type PeerConfig struct {
	Port          int
	EclipsedCid   *string
	Ip            *string
	Identity      config.Identity
	SybilFilePath *string
}

func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

func createTempRepo(peerConfig PeerConfig, otherPeers []multiaddr.Multiaddr) (string, error) {
	repoPath, err := os.MkdirTemp("", "ipfs-shell")
	if err != nil {
		return "", fmt.Errorf("failed to get temp dir: %s", err)
	}

	// Create a config with default options and a 2048 bit key
	var cfg *config.Config
	if peerConfig.Identity.PrivKey == "" {
		cfg, err = config.Init(io.Discard, 2048)
	} else {
		cfg, err = config.InitWithIdentity(peerConfig.Identity)
	}
	if err != nil {
		return "", err
	}

	// Set only ip4 addresses because ip6 causes problem to diversity filter
	cfg.Addresses.Swarm = []string{
		fmt.Sprintf("/ip4/%s/tcp/%d", *peerConfig.Ip, peerConfig.Port),
		fmt.Sprintf("/ip4/%s/udp/%d/quic-v1", *peerConfig.Ip, peerConfig.Port),
		fmt.Sprintf("/ip4/%s/udp/%d/quic-v1/webtransport", *peerConfig.Ip, peerConfig.Port),
	}

	// Allow all the other peers to connect to this one
	for _, otherPeer := range otherPeers {
		cfg.Swarm.ResourceMgr.Allowlist = append(cfg.Swarm.ResourceMgr.Allowlist, otherPeer.String())
	}

	cfg.Peering.Peers, err = peer.AddrInfosFromP2pAddrs(otherPeers...)
	if err != nil {
		fmt.Println("failed to parse multiaddrs of other peers")
		panic(err)
	}

	// Create the repo with the config
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to init ephemeral node: %s", err)
	}

	return repoPath, nil
}

// Creates an IPFS node and returns its coreAPI.
func createNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		Repo:    repo,
	}

	return core.NewNode(ctx, nodeOptions)
}

var loadPluginsOnce sync.Once

// Spawns a node to be used just for this run (i.e. creates a tmp repo).
func SpawnEphemeral(ctx context.Context, peerConfig PeerConfig, otherPeers []multiaddr.Multiaddr) (coreiface.CoreAPI, *core.IpfsNode, error) {
	var onceErr error
	loadPluginsOnce.Do(func() {
		onceErr = setupPlugins("")
	})
	if onceErr != nil {
		return nil, nil, onceErr
	}

	// Create a Temporary Repo
	repoPath, err := createTempRepo(peerConfig, otherPeers)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp repo: %s", err)
	}

	node, err := createNode(ctx, repoPath)
	if err != nil {
		return nil, nil, err
	}

	api, err := coreapi.NewCoreAPI(node)

	fmt.Println("Peer is UP: "+node.Identity.String(), "\n")

	return api, node, err
}
