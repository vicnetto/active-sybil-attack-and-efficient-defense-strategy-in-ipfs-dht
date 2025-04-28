module distribution-kl-from-ns

go 1.23.2

toolchain go1.24.2

// Use branch detection-and-region-based-queries
// replace github.com/libp2p/go-libp2p-kad-dht => github.com/vicnetto/go-libp2p-kad-dht detection-and-region-based-queries
replace github.com/libp2p/go-libp2p-kad-dht => github.com/vicnetto/go-libp2p-kad-dht v0.28.3-0.20250428203719-9990d029ef36

replace github.com/ssrivatsan97/go-libp2p-kad-dht/eclipse-detection => github.com/vicnetto/go-libp2p-kad-dht/eclipse-detection v0.0.0-20250428203719-9990d029ef36

require (
	github.com/ssrivatsan97/go-libp2p-kad-dht/eclipse-detection v0.0.0-20230716010606-8042c7659c79
	github.com/vicnetto/active-sybil-attack/logger v0.0.0-20250120102219-254a63da6821
	github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl v0.0.0-20250120102219-254a63da6821
)

require (
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/ipfs/boxo v0.8.1 // indirect
	github.com/ipfs/go-cid v0.4.1 // indirect
	github.com/ipfs/go-log v1.0.5 // indirect
	github.com/ipfs/go-log/v2 v2.5.1 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/libp2p/go-buffer-pool v0.1.0 // indirect
	github.com/libp2p/go-cidranger v1.1.0 // indirect
	github.com/libp2p/go-libp2p v0.27.6 // indirect
	github.com/libp2p/go-libp2p-asn-util v0.3.0 // indirect
	github.com/libp2p/go-libp2p-kbucket v0.6.3 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.2.0 // indirect
	github.com/multiformats/go-multiaddr v0.9.0 // indirect
	github.com/multiformats/go-multibase v0.2.0 // indirect
	github.com/multiformats/go-multicodec v0.9.0 // indirect
	github.com/multiformats/go-multihash v0.2.3 // indirect
	github.com/multiformats/go-multistream v0.4.1 // indirect
	github.com/multiformats/go-varint v0.0.7 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	golang.org/x/crypto v0.9.0 // indirect
	golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa // indirect
	golang.org/x/sys v0.14.0 // indirect
	gonum.org/v1/gonum v0.15.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	lukechampine.com/blake3 v1.2.1 // indirect
)
