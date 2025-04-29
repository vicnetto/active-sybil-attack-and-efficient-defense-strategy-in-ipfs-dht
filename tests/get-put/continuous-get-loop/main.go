package main

import (
	"flag"
	"fmt"
	"github.com/vicnetto/active-sybil-attack/logger"
	"github.com/vicnetto/active-sybil-attack/utils/pid-generation/generate"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

var log = logger.InitializeLogger()

const outDir = "out"
const maxIntervalInMinutes = 60*24*7 + 1

// Worker manages multiple instances of the same executable
type Worker struct {
	executable string
	args       []string
	cmd        *exec.Cmd
}

type QuantityPerInterval struct {
	restartTime time.Duration
	quantity    int
}

type Flags struct {
	getLoopPath            string
	cidFilepath            string
	peerFilepath           string
	providerPid            string
	quantityPerRestartTime []QuantityPerInterval
	interval               time.Duration
	port                   int
}

func help() func() {
	return func() {
		log.Info.Println("\nUsage:", os.Args[0], "[flags]:")
		log.Info.Println("  -privateKey <string>  -- Private key of the test node")
		log.Info.Println("  -port <int>           -- Port to run the test node")
		log.Info.Println("  -cidFilepath <string> -- CIDs to be tested for each test")
		log.Info.Println("  -providerPid <string> -- Peer ID of the provider")
		log.Info.Println("	-<int> <int>          -- Specify the quantity of nodes for each restart time. Multiple restart times can be specified.")
		log.Info.Println("	                         Example: -120 2 -1440 2")
		log.Info.Println("	                                  |      *-----> Restart time: 1440m, quantity of nodes: 2")
		log.Info.Println("	                                  *------------> Restart time: 120m, quantity of nodes: 2")
	}
}

func treatFlags() Flags {
	flags := Flags{}
	var interval int
	flag.StringVar(&flags.getLoopPath, "getLoopPath", "", "")
	flag.StringVar(&flags.peerFilepath, "peerFilepath", "", "")
	flag.StringVar(&flags.cidFilepath, "cidFilepath", "", "")
	flag.StringVar(&flags.providerPid, "providerPid", "", "")
	flag.IntVar(&interval, "interval", 1, "")
	flag.IntVar(&flags.port, "port", 10000, "")

	var quantityPerRestartTime [maxIntervalInMinutes]int
	for i := 0; i < maxIntervalInMinutes; i++ {
		flag.IntVar(&quantityPerRestartTime[i], strconv.Itoa(i), 0, "")
	}

	flag.Usage = help()
	flag.Parse()

	missingFlag := false

	flags.quantityPerRestartTime = []QuantityPerInterval{}
	for i := maxIntervalInMinutes - 1; i > 0; i-- {
		if quantityPerRestartTime[i] > 0 {
			flags.quantityPerRestartTime = append(flags.quantityPerRestartTime,
				QuantityPerInterval{quantity: quantityPerRestartTime[i], restartTime: time.Duration(i) * time.Minute})
		}
	}

	if len(flags.getLoopPath) == 0 {
		log.Info.Println("error: flag getLoopPath missing.")
		missingFlag = true
	}

	if len(flags.peerFilepath) == 0 {
		log.Info.Println("error: flag peerFilepath missing.")
		missingFlag = true
	}

	if len(flags.cidFilepath) == 0 {
		log.Info.Println("error: flag cidFilepath missing.")
		missingFlag = true
	}

	if len(flags.providerPid) == 0 {
		log.Info.Println("error: flag providerPid missing.")
		missingFlag = true
	}

	flags.interval = time.Duration(interval) * time.Minute

	if missingFlag {
		flag.Usage()
		os.Exit(1)
	}

	return flags
}

// NewWorker initializes the Worker with executable and args
func NewWorker(executable string, args []string) *Worker {
	return &Worker{
		executable: executable,
		args:       args,
		cmd:        nil,
	}
}

// run launches a single process with the specified id
func (pm *Worker) run(peer string, restartTime int) {
	iteration := 0

	for {
		iteration++
		// Open the file in append mode, create it if it doesn't exist
		outPath := filepath.Join(outDir, fmt.Sprintf("%d-%s.out", restartTime, peer))
		outfile, err := os.OpenFile(outPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Error.Printf("could not open or create file %s: %s", outPath, err)
		}

		// Start the process
		cmd := exec.Command(pm.executable, append(pm.args, "-iteration", fmt.Sprint(iteration))...)
		cmd.Stdout = outfile
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		pm.cmd = cmd

		err = cmd.Start()
		if err != nil {
			log.Error.Printf("Iteration %d) Error starting process in peer %s: %v\n", iteration, peer, err)
			time.Sleep(5 * time.Second) // Retry delay
			continue
		}
		log.Info.Printf("Iteration %d) Starting lookups with peer %s: PID %d\n", iteration, peer, cmd.Process.Pid)

		// Wait for the process to finish
		err = cmd.Wait()
		if err != nil {
			log.Error.Printf("Iteration %d) Finished with error in peer %s: PID %v\n", iteration, peer, err)
			time.Sleep(5 * time.Second) // Retry delay
			continue
		} else {
			log.Info.Printf("Iteration %d) Finished normally in peer %s.\n", iteration, peer)
		}

		outfile.Close()

		// Restart the process after a short delay
		time.Sleep(1 * time.Second)
	}
}

// terminateAll kills all processes that are still running
func (pm *Worker) terminate() {
	fmt.Println()
	if pm.cmd.Process != nil {
		log.Info.Printf("Killing process with PID: %d\n", pm.cmd.Process.Pid)
		syscall.Kill(-pm.cmd.Process.Pid, syscall.SIGTERM)
	}
}

func main() {
	flags := treatFlags()
	peers := generate.ReadAndFormatPeers(flags.peerFilepath)

	if err := os.MkdirAll(outDir, os.ModePerm); err != nil {
		log.Error.Printf("could create the log directory: %s", err)
		return
	}

	necessaryNodes := 0
	for _, quantityPerRestart := range flags.quantityPerRestartTime {
		necessaryNodes += quantityPerRestart.quantity
	}

	if necessaryNodes > len(peers) {
		log.Error.Printf("More nodes required than available in the nodes file.")
		return
	}

	for i, interval := range flags.quantityPerRestartTime {
		log.Info.Printf("Interval %d -- restart every %s:", i+1, interval.restartTime)
		log.Info.Printf(" Quantity of nodes: %d", interval.quantity)
		log.Info.Printf(" Verifications between each start: %d", int(interval.restartTime.Minutes()/flags.interval.Minutes()))
	}

	var manager []*Worker
	currentPeer := 0
	startInterval := time.Duration(2)*time.Minute + time.Duration(30)*time.Second
	if int(startInterval.Minutes()) == 0 {
		startInterval = time.Duration(1) * time.Minute
	}
	log.Info.Println("Interval between the launch of two peers:", startInterval)

	for _, quantityPerRestart := range flags.quantityPerRestartTime {
		quantity := quantityPerRestart.quantity

		for quantity > 0 {
			peer := peers[currentPeer]
			var args []string
			args = append(args, "-cidFilepath", flags.cidFilepath,
				"-providerPid", flags.providerPid,
				"-interval", fmt.Sprint(int(flags.interval.Minutes())),
				"-verifications", fmt.Sprint(int(quantityPerRestart.restartTime.Minutes()/flags.interval.Minutes())),
				"-port", fmt.Sprint(peer.Port),
				"-privateKey", peer.PrivateKey)

			pm := NewWorker(flags.getLoopPath, args)
			manager = append(manager, pm)

			go pm.run(peer.PeerID, int(quantityPerRestart.restartTime.Minutes()))
			log.Info.Println("Sleeping for", startInterval, "to launch next test")
			time.Sleep(startInterval)

			currentPeer++
			quantity--
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	for _, pm := range manager {
		pm.terminate()
	}

}
