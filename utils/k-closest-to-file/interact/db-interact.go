package interact

import (
	"bufio"
	"fmt"
	gocid "github.com/ipfs/go-cid"
	kspace "github.com/libp2p/go-libp2p-kbucket/keyspace"
	"github.com/libp2p/go-libp2p/core/peer"
	mh "github.com/multiformats/go-multihash"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
)

var k = 20

// StoreDHTLookupToFile stores a DHT lookup result into a file.
// The file is created in the specified relative destination folder, which will be created if it does not exist.
func StoreDHTLookupToFile(cid gocid.Cid, peers []peer.ID, relativePath string) error {
	// Ensure the destination folder exists or create it.
	if err := os.MkdirAll(relativePath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create destination folder: %w", err)
	}

	// Define the full path to the output file.
	filePath := filepath.Join(relativePath, cid.String())

	// Create the file for writing.
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write each peer ID to the file, one per line.
	writer := bufio.NewWriter(file)
	for _, peerID := range peers {
		if _, err := writer.WriteString(peerID.String() + "\n"); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
	}

	// Ensure all buffered data is flushed to the file.
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer: %w", err)
	}

	return nil
}

// GetRandomDHTLookup retrieves one random DHT lookup from the specified folder.
// Returns the data as a map[gocid.Cid][]peer.ID.
func GetRandomDHTLookup(relativePath string) (gocid.Cid, []peer.ID, error) {
	lookups, err := GetRandomDHTLookups(1, relativePath)

	var cid gocid.Cid
	var pid []peer.ID

	for contentId, ids := range lookups {
		cid = contentId
		pid = ids
	}

	return cid, pid, err
}

// GetRandomDHTLookups retrieves $n random DHT lookups from the specified folder.
// Returns the data as a map[gocid.Cid][]peer.ID.
func GetRandomDHTLookups(n int, relativePath string) (map[gocid.Cid][]peer.ID, error) {
	// Open the folder.
	dirEntries, err := os.ReadDir(relativePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Filter the entries to files only and shuffle them.
	fileEntries := []os.DirEntry{}
	for _, entry := range dirEntries {
		if entry.Type().IsRegular() {
			fileEntries = append(fileEntries, entry)
		}
	}

	// Prepare the result map.
	result := make(map[gocid.Cid][]peer.ID)

	for i := 0; i < n && len(fileEntries) != 0; i++ {
		// Get random file and remove it from file
		randomPos := rand.Intn(len(fileEntries))
		file := fileEntries[randomPos]
		fileEntries = append(fileEntries[:randomPos], fileEntries[randomPos+1:]...)

		filePath := filepath.Join(relativePath, file.Name())

		// Parse the CID from the filename.
		c, err := gocid.Parse(file.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to parse CID from file name '%s': %w", file.Name(), err)
		}

		// Open the file for reading.
		f, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file '%s': %w", filePath, err)
		}

		// Read the peer IDs from the file.
		var peers []peer.ID
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			peerID, err := peer.Decode(scanner.Text())
			if err != nil {
				return nil, fmt.Errorf("failed to decode peer ID from file '%s': %w", filePath, err)
			}
			peers = append(peers, peerID)
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read file '%s': %w", filePath, err)
		}

		// Add to the result map.
		result[c] = peers

		f.Close()
	}

	return result, nil
}

// GetClosestKFromContactedPeers sorts the contactedPeers list and returns the k closest peers
func GetClosestKFromContactedPeers(cid gocid.Cid, contactedPeers []peer.ID) ([]peer.ID, error) {
	closest, err := SortByDistance(cid, contactedPeers)
	if err != nil {
		return nil, err
	}

	closest = closest[:k-1]

	return closest, nil
}

func SortByDistance(cid gocid.Cid, peers []peer.ID) ([]peer.ID, error) {
	targetCIDByte, _ := mh.FromB58String(cid.String())
	targetCIDKey := kspace.XORKeySpace.Key(targetCIDByte)

	distanceCmp := func(a, b peer.ID) int {
		aMultiHash, _ := mh.FromB58String(a.String())
		aPeerKey := kspace.XORKeySpace.Key(aMultiHash)
		aDistance := aPeerKey.Distance(targetCIDKey)

		bMultiHash, _ := mh.FromB58String(b.String())
		bPeerKey := kspace.XORKeySpace.Key(bMultiHash)
		bDistance := bPeerKey.Distance(targetCIDKey)

		return aDistance.Cmp(bDistance)
	}

	slices.SortFunc(peers, distanceCmp)

	return peers, nil
}
