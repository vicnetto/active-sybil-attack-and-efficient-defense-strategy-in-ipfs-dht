package generate

import "time"

var Timeout = 120 * time.Second

const MaxProbabilities = 40

type NodePerCpl struct {
	Cpl      int
	Quantity int
}

type PidGenerateConfig struct {
	Quantity   int
	Cid        string
	FirstPort  int
	OutFile    string
	UseAllCpus bool

	ByInterval bool
	FirstPeer  string
	SecondPeer string

	ByClosest   bool
	ByCpl       bool
	Cpl         int
	NodesPerCpl []NodePerCpl

	ByBase32      bool
	ReferencePeer string
}
