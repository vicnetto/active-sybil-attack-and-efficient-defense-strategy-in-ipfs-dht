package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ipfs/kubo/core"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/sr"
	srutils "github.com/libp2p/go-libp2p-kad-dht/sr/utils"
	"github.com/vicnetto/active-sybil-attack/logger"
	ipfspeer "github.com/vicnetto/active-sybil-attack/node/peer"
	"github.com/vicnetto/active-sybil-attack/utils/k-closest-to-file/interact"
	"math"
	"os"
	"syscall"
	"time"
)

var log = logger.InitializeLogger()
var dbPath = "../../../db"

type FlagConfig struct {
	estimationTests int
	estimationPeers int
	perfectingPeers int
	perfectingTests int
	port            int
	privateKey      string
}

func help() func() {
	return func() {
		fmt.Println("Usage:", os.Args[0], "[flags]:")
		fmt.Println("  -estimationTests <int> -- Quantity of tests to a random CID for the first distance estimation")
		fmt.Println("  -estimationPeers <int> -- Max peers to contact for obtaining the distance average")
		fmt.Println("  -perfectingTests <int> -- Quantity of tests to a random CID for perfecting the distance estimation")
		fmt.Println("  -perfectingPeers <int> -- Max peers to contact for perfecting the distance average in a second moment")
		fmt.Println("  -port <int>            -- Port of the IPFS node. (default: any valid port)")
		fmt.Println("  -privateKey <string>   -- Private key of the IPFS node. (default: random node)")
	}
}

func treatFlags() *FlagConfig {
	flagConfig := FlagConfig{}

	flag.IntVar(&flagConfig.estimationTests, "estimationTests", 0, "")
	flag.IntVar(&flagConfig.estimationPeers, "estimationPeers", 0, "")
	flag.IntVar(&flagConfig.perfectingTests, "perfectingTests", 0, "")
	flag.IntVar(&flagConfig.perfectingPeers, "perfectingPeers", 0, "")
	flag.IntVar(&flagConfig.port, "port", 0, "")
	flag.StringVar(&flagConfig.privateKey, "privateKey", "", "")

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	if flagConfig.estimationTests == 0 {
		log.Error.Println("error: flag estimationTests missing.")
		missingFlag = true
	}

	if flagConfig.estimationPeers == 0 {
		log.Error.Println("error: flag estimationPeers missing.")
		missingFlag = true
	}

	if flagConfig.perfectingTests == 0 {
		log.Error.Println("error: flag perfectingTests missing.")
		missingFlag = true
	}

	if flagConfig.perfectingPeers == 0 {
		log.Error.Println("error: flag perfectingPeers missing.")
		missingFlag = true
	}

	if missingFlag {
		log.Error.Println()
		flag.Usage()
		os.Exit(1)
	}

	return &flagConfig
}

func calculateAveragePerMethod(peersPerDistance []map[sr.WelfordAverage]srutils.QuantityPerAverage, distance sr.WelfordAverage, method sr.MeanType) int {
	var average, quantity float64

	for _, cidResult := range peersPerDistance {
		perMethod := cidResult[distance]

		quantity++

		average += float64(perMethod[method])
	}

	return int(math.Round(average / quantity))
}

func peersQuantityPerAverageString(distance sr.WelfordAverage, peersPerDistance map[sr.WelfordAverage]srutils.QuantityPerAverage) string {
	var toPrint string

	for mt := sr.MeanType(0); mt <= sr.LastMeanType; mt++ {
		toPrint += fmt.Sprintf(" %s: %d (%s);", mt.String(), peersPerDistance[distance][mt],
			dht.ToSciNotation(distance.GetAverage(mt)))
	}

	return toPrint
}

func peersQuantityPerAverageOfAllTestsString(maxDistance sr.WelfordAverage,
	peersPerDistance []map[sr.WelfordAverage]srutils.QuantityPerAverage, csv bool) string {
	var toPrint string
	for mt := sr.MeanType(0); mt <= sr.LastMeanType; mt++ {
		if csv {
			toPrint += fmt.Sprintf("%s;%d;", dht.ToSciNotation(maxDistance.GetAverage(mt)),
				calculateAveragePerMethod(peersPerDistance, maxDistance, mt))
		} else {
			toPrint += fmt.Sprintf("%s: %d (%s); ", mt.String(), calculateAveragePerMethod(peersPerDistance, maxDistance, mt),
				dht.ToSciNotation(maxDistance.GetAverage(mt)))
		}
	}

	return toPrint
}

func peersQuantityPerAverageIndividualCsvString(maxDistance sr.WelfordAverage,
	peersPerDistance map[sr.WelfordAverage]srutils.QuantityPerAverage) string {

	var toPrint string
	for mt := sr.MeanType(0); mt <= sr.LastMeanType; mt++ {
		toPrint += fmt.Sprintf("%s;%d;", dht.ToSciNotation(maxDistance.GetAverage(mt)),
			peersPerDistance[maxDistance][mt])
	}
	return toPrint
}

func performLookupToImproveResultUsingDb(lookupCount int, average sr.WelfordAverage,
	dbPath string) ([]map[sr.WelfordAverage]srutils.QuantityPerAverage, error) {
	var peersPerDistance []map[sr.WelfordAverage]srutils.QuantityPerAverage

	log.Info.Printf("Obtaining random lookups from the db...")
	lookups, err := interact.GetRandomDHTLookups(lookupCount, dbPath)
	if err != nil {
		return nil, err
	}

	var i int
	for cid, contactedPeers := range lookups {
		peersPerDistance = append(peersPerDistance, make(map[sr.WelfordAverage]srutils.QuantityPerAverage))
		peersPerDistance[i][average] = srutils.NewQuantityPerAverage()

		peers, err := interact.GetClosestKFromContactedPeers(cid, contactedPeers)
		if err != nil {
			return nil, err
		}

		srutils.CountPeersPerAverage(cid, []sr.WelfordAverage{average}, peers, &peersPerDistance[i])
		log.Info.Printf("   %s: %s", cid.String(), peersQuantityPerAverageString(average, peersPerDistance[i]))
		i++
	}

	return peersPerDistance, nil
}

func performLookupToVerifyDistance(flagConfig FlagConfig, ctx context.Context, clientNode *core.IpfsNode,
	average sr.WelfordAverage) []map[sr.WelfordAverage]srutils.QuantityPerAverage {
	var peersPerDistance []map[sr.WelfordAverage]srutils.QuantityPerAverage

	for i := 0; i < flagConfig.perfectingTests; i++ {
		peersPerDistance = append(peersPerDistance, make(map[sr.WelfordAverage]srutils.QuantityPerAverage))
		peersPerDistance[i][average] = srutils.NewQuantityPerAverage()

		log.Info.Printf("%d) Performing random lookups to verify the average distances calculated:", i+1)
		cidDecode, contactedPeers := srutils.PerformRandomLookupReturningAllQueriedPeers(ctx, clientNode)

		srutils.CountPeersPerAverage(cidDecode, []sr.WelfordAverage{average}, contactedPeers, &peersPerDistance[i])
		log.Info.Printf("    %s", peersQuantityPerAverageString(average, peersPerDistance[i]))
	}

	return peersPerDistance
}

func performLookupToVerifyDistanceUsingDb(lookupCount int, average sr.WelfordAverage,
	dbPath string) ([]map[sr.WelfordAverage]srutils.QuantityPerAverage, error) {
	var peersPerDistance []map[sr.WelfordAverage]srutils.QuantityPerAverage

	log.Info.Printf("Obtaining random lookups from the db...")
	lookups, err := interact.GetRandomDHTLookups(lookupCount, dbPath)
	if err != nil {
		return nil, err
	}

	var i int
	for cid, contactedPeers := range lookups {
		peersPerDistance = append(peersPerDistance, make(map[sr.WelfordAverage]srutils.QuantityPerAverage))
		peersPerDistance[i][average] = srutils.NewQuantityPerAverage()

		srutils.CountPeersPerAverage(cid, []sr.WelfordAverage{average}, contactedPeers, &peersPerDistance[i])

		log.Info.Printf("   %s: %s", cid.String(), peersQuantityPerAverageString(average, peersPerDistance[i]))

		i++
	}

	return peersPerDistance, nil
}

func testQuery(ctx context.Context, flagConfig FlagConfig, clientNode *core.IpfsNode, quantity int) {
	// var alreadyQuried []peer.ID
	var estimationAverageAll []sr.WelfordAverage
	var estimationPeersPerDistanceAll [][]map[sr.WelfordAverage]srutils.QuantityPerAverage

	for i := 0; i < quantity; i++ {
		estimationAverage, err := clientNode.DHT.WAN.GetFarthestKAverageByQuery(ctx, flagConfig.estimationPeers)
		if err != nil {
			panic(err)
		}

		estimationPeersPerDistance, err := performLookupToVerifyDistanceUsingDb(flagConfig.estimationTests, estimationAverage, dbPath)
		if err != nil {
			panic(err)
		}

		log.Info.Println("Result before improving:")
		log.Info.Printf("  %d peers) %s", flagConfig.estimationPeers, peersQuantityPerAverageOfAllTestsString(estimationAverage, estimationPeersPerDistance, false))

		fmt.Println()
		log.Info.Println("**CSV export**")
		fmt.Println("time;testId;mean;peers[mean];meanWithStdDev;peers[meanWithStdDev];weightedMean;peers[weightedMean];weightedWithStdDev;peers[weightedWithStdDev];cpl;peers[cpl]")
		for i := 0; i < flagConfig.estimationTests; i++ {
			fmt.Printf("BEFORE;%d;%s\n", i+1, peersQuantityPerAverageIndividualCsvString(estimationAverage, estimationPeersPerDistance[i]))
		}

		estimationAverageAll = append(estimationAverageAll, estimationAverage)
		estimationPeersPerDistanceAll = append(estimationPeersPerDistanceAll, estimationPeersPerDistance)
	}

	fmt.Println()
	log.Info.Println("**Average CSV export**")
	fmt.Println("time;mean;peers[mean];meanWithStdDev;peers[meanWithStdDev];weightedMean;peers[weightedMean];weightedWithStdDev;peers[weightedWithStdDev];cpl;peers[cpl]")
	for i := 0; i < quantity; i++ {
		fmt.Printf("BEFORE;%s\n", peersQuantityPerAverageOfAllTestsString(estimationAverageAll[i], estimationPeersPerDistanceAll[i], true))
	}

	clientNode.Close()

	syscall.Exit(0)
}

func main() {
	flagConfig := treatFlags()

	dht.Debug = true

	ctx, cancel := context.WithCancel(context.Background())

	var peerConfig ipfspeer.Config
	if len(flagConfig.privateKey) != 0 {
		peerConfig = ipfspeer.ConfigForSpecificNode(flagConfig.port, flagConfig.privateKey)
	} else {
		peerConfig = ipfspeer.ConfigForRandomNode(flagConfig.port)
	}

	_, clientNode, err := ipfspeer.SpawnEphemeral(ctx, peerConfig)
	if err != nil {
		panic(err)
	}
	log.Info.Println("PID is UP:", clientNode.Identity.String())

	log.Info.Println("Sleep for 10 seconds before starting...")
	time.Sleep(10 * time.Second)

	// testQuery(ctx, *flagConfig, clientNode, 10)

	estimationAverage, err := clientNode.DHT.WAN.GetFarthestKAverageByQuery(ctx, flagConfig.estimationPeers)
	if err != nil {
		panic(err)
	}

	estimationPeersPerDistance, err := performLookupToVerifyDistanceUsingDb(flagConfig.estimationTests, estimationAverage, dbPath)
	if err != nil {
		panic(err)
	}

	log.Info.Println("Result before improving:")
	log.Info.Printf("  %d peers) %s", flagConfig.estimationPeers, peersQuantityPerAverageOfAllTestsString(estimationAverage, estimationPeersPerDistance, false))

	log.Info.Printf("Perfecting the distance previously calculated by asking %d peers...", flagConfig.perfectingPeers)
	perfectingAverage, err := clientNode.DHT.WAN.GetFarthestKAverageByLookup(ctx, flagConfig.perfectingPeers, &estimationAverage)
	if err != nil {
		panic(err)
	}

	perfectingPeersPerDistance, err := performLookupToVerifyDistanceUsingDb(flagConfig.perfectingTests, perfectingAverage, dbPath)
	if err != nil {
		panic(err)
	}

	log.Info.Println("Result after improving the distance average:")
	log.Info.Printf("  %d (%d + %d) peers) %s", flagConfig.estimationPeers+flagConfig.perfectingPeers,
		flagConfig.estimationPeers, flagConfig.perfectingPeers,
		peersQuantityPerAverageOfAllTestsString(perfectingAverage, perfectingPeersPerDistance, false))

	fmt.Println()
	log.Info.Println("**CSV export**")
	fmt.Println("time;testId;mean;peers[mean];meanWithStdDev;peers[meanWithStdDev];weightedMean;peers[weightedMean];weightedWithStdDev;peers[weightedWithStdDev];cpl;peers[cpl]")
	for i := 0; i < flagConfig.estimationTests; i++ {
		fmt.Printf("BEFORE;%d;%s\n", i+1, peersQuantityPerAverageIndividualCsvString(estimationAverage, estimationPeersPerDistance[i]))
	}
	for i := 0; i < flagConfig.perfectingTests; i++ {
		fmt.Printf("AFTER;%d;%s\n", i+1, peersQuantityPerAverageIndividualCsvString(perfectingAverage, perfectingPeersPerDistance[i]))
	}

	fmt.Println()
	log.Info.Println("**Average CSV export**")
	fmt.Println("time;mean;peers[mean];meanWithStdDev;peers[meanWithStdDev];weightedMean;peers[weightedMean];weightedWithStdDev;peers[weightedWithStdDev];cpl;peers[cpl]")
	fmt.Printf("BEFORE;%s\n", peersQuantityPerAverageOfAllTestsString(estimationAverage, estimationPeersPerDistance, true))
	fmt.Printf("AFTER;%s\n", peersQuantityPerAverageOfAllTestsString(perfectingAverage, perfectingPeersPerDistance, true))

	clientNode.Close()
	cancel()
}
