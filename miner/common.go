package miner

import (
	"../shared"
	"crypto/ecdsa"
	"net"
	"net/rpc"
	"sync"
	"time"
)

// Settings for an instance of the BlockArt project/network.
type MinerNetSettings struct {
	// Hash of the very first (empty) block in the chain.
	GenesisBlockHash string

	// The minimum number of ink miners that an ink miner should be
	// connected to. If the ink miner dips below this number, then
	// they have to retrieve more nodes from the server using
	// GetNodes().
	MinNumMinerConnections uint8

	// Mining ink reward per op and no-op blocks (>= 1)
	InkPerOpBlock   uint32
	InkPerNoOpBlock uint32

	// Number of milliseconds between heartbeat messages to the server.
	HeartBeat uint32

	// Proof of work difficulty: number of zeroes in prefix (>=0)
	PoWDifficultyOpBlock   uint8
	PoWDifficultyNoOpBlock uint8

	// Canvas settings
	CanvasSettings shared.CanvasSettings
}

// Type for storing all canvas info
type Canvas struct {
	Shapes     map[string]Shape // can also be []Shape
	XMax, YMax uint32
}

var (
	Log = shared.NewLogger(true, true, true)

	ServerIPPort string
	Server       *rpc.Client
	MinerIPPort  string // IP:Port for miner-miner

	PrivKey   *ecdsa.PrivateKey
	PubKeyStr string

	NetSettings MinerNetSettings

	ActiveNeighbours map[string]*rpc.Client
	NeighbourLock    *sync.Mutex
	NumNeighbours    int
	CheckInterval    time.Duration

	// Indicate whether the block chain is successfully initialized
	initialized bool
)

// Asynchronous, caller needs to call Wait()
func InitRPCServer(v interface{}) (listener net.Listener, err error) {
	Log.Trace(v)

	// Get local ip
	serverAddr, err := getOutBoundIP()
	if err != nil {
		return
	}

	rpc.Register(v)

	// set up RPC listener
	listener, err = net.Listen("tcp", serverAddr+":0")
	if err != nil {
		return
	}

	localIPPort = listener.Addr().String()
	Log.Debug("local address %s", localIPPort)

	// Start RPC Server
	go func() {
		Log.Debug("RPC Server for artnode ready")
		rpc.Accept(listener)
	}()

	return
}

// Get the outward facing interface IP of the machine
func getOutBoundIP() (string, error) {
	conn, err := net.Dial("udp", "1.1.1.1:1111")
	if err != nil {
		return "", err
	}

	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func init() {
	NetSettings.CanvasSettings = shared.CanvasSettings{}
	NumNeighbours = 0
	NeighbourLock = new(sync.Mutex)
	ActiveNeighbours = make(map[string]*rpc.Client)
	CheckInterval = 2000 * time.Millisecond
	artnodeRegHash = shared.HashByteArr([]byte(shared.ArtnodeRegMsg))
}
