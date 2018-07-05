package main

import (
	"./miner"
	"./shared"
	"crypto/elliptic"
	"encoding/gob"
	"net"
	"net/rpc"
	"os"
	"time"
)

// go run ink-miner.go [server ip:port] [pubKey] [privKey]
func main() {
	args := os.Args[1:]
	miner.Log.Trace(args)

	err := parseArgs(args)
	if err != nil {
		os.Exit(1)
	}

	listener, err := miner.InitRPCServer(new(miner.MinerMinerRPC))
	if err != nil {
		miner.Log.Error("Failed to set up rpc between miners [%s]", err.Error())
		os.Exit(1)
	}
	defer listener.Close()
	miner.MinerIPPort = listener.Addr().String()

	serverConn, err := contactServer(miner.MinerIPPort)
	if err != nil {
		miner.Log.Error("Failed to register with server [%s]", err.Error())
		os.Exit(1)
	}
	miner.Log.Debug("Received settings from server [%v]", miner.NetSettings)
	miner.Server = serverConn

	HeartBeatLoop()
	ConnectToMiners()

	miner.Log.Debug("Num neighbours initially connected to: [%v]", miner.NumNeighbours)
	CheckNeighbours()

	// create a thread for artnode-miner rpc server
	_, err = miner.InitRPCServer(new(miner.MinerArtRPC))
	if err != nil {
		miner.Log.Error("Failed to set up rpc between artnode and miner [%s]", err.Error())
		os.Exit(1)
	}

	// Initialize miner-miner
	err, done := miner.InitBlockchain()
	if err != nil {
		miner.Log.Error("Failed to initialize miner blockchain", err.Error())
		os.Exit(1)
	}

	miner.Log.Debug("Miner ready")

	// Miner mines endlessly, this variable stops main from exiting
	// done will never be invoked.
	<-done

	miner.Log.Trace("Miner exiting")
}

func contactServer(serverAddr string) (server *rpc.Client, err error) {
	gob.Register(&net.TCPAddr{})
	gob.Register(&elliptic.CurveParams{})

	minerAddr, err := net.ResolveTCPAddr("tcp", miner.MinerIPPort)
	server, err = rpc.Dial("tcp", miner.ServerIPPort)
	if err != nil {
		return
	}

	registerArgs := shared.RegisterArgs{Address: minerAddr, Key: miner.PrivKey.PublicKey}
	miner.Log.Debug("Registering with: [%v] [%v]", minerAddr, miner.PrivKey.PublicKey)
	err = server.Call("RServer.Register", registerArgs, &miner.NetSettings)
	if err != nil {
		return
	}
	return
}

func SendHeartBeat() (err error) {
	err = miner.Server.Call("RServer.HeartBeat", miner.PrivKey.PublicKey, nil)
	if err != nil {
		// try re-registering with the server
		miner.Log.Trace("Failed to send heartbeat to server. Reregistering...")
		minerAddr, _ := net.ResolveTCPAddr("tcp", miner.MinerIPPort)
		registerArgs := shared.RegisterArgs{Address: minerAddr, Key: miner.PrivKey.PublicKey}
		err = miner.Server.Call("RServer.Register", registerArgs, nil)
		return
	}
	time.Sleep(time.Duration(miner.NetSettings.HeartBeat) * 500 * time.Microsecond)
	return
}

func HeartBeatLoop() {
	go func() {
		for {
			err := SendHeartBeat()
			if err != nil {
				miner.Log.Error("Could not send heartbeats to server [%s]", err.Error())
			}
		}
	}()
}

func ConnectToMiners() (err error) {
	minerInfo := shared.ConnectMinerArgs{IPPort: miner.MinerIPPort, EncodedPubKey: miner.PubKeyStr}
	var minerAddrs []net.Addr

	err = miner.Server.Call("RServer.GetNodes", miner.PrivKey.PublicKey, &minerAddrs)
	if err != nil {
		miner.Log.Error("Error from server [%s]", err.Error())
		return
	}

	for _, addr := range minerAddrs {
		if conn, connErr := rpc.Dial("tcp", addr.String()); connErr == nil {
			var minerPubKey string
			c := make(chan error, 1)
			c <- conn.Call("MinerMinerRPC.Connect", minerInfo, &minerPubKey)
			select {
			case connErr := <-c:
				if connErr == nil && minerPubKey != "" {
					miner.UpdateNeighbours(minerPubKey, conn)
				}
				return
			case <-time.After(100 * time.Millisecond):
				return
			}
		}
	}
	return
}

// Check on all the neighbours every 5 seconds
func CheckNeighbours() {
	reply := new(bool)
	minConns := miner.NetSettings.MinNumMinerConnections
	go func() {
		for {
			for pKey, conn := range miner.ActiveNeighbours {
				err := conn.Call("MinerMinerRPC.IsAlive", 0, reply)
				if err != nil {
					miner.RemoveInactiveMiner(pKey)
				}
				// Check that we still have enough ink miners
				if miner.NumNeighbours < int(minConns) {
					err = ConnectToMiners()
					if err != nil {
						miner.Log.Error("Cannot contact server to get new miners [%s]", err.Error())
					}
				}
			}
			time.Sleep(miner.CheckInterval)
		}
	}()
}

func parseArgs(args []string) (err error) {
	if len(args) != 3 {
		miner.Log.Error("invalid number of arguments\nusage: $ go run ink-miner.go [server ip:port] [pubKey] [privKey]")
		err = os.ErrInvalid
		return
	}

	// Get args
	miner.ServerIPPort = args[0]
	miner.PrivKey, err = shared.DecodePrivKey(args[2])
	if err != nil {
		miner.Log.Error("Failed to parse key pairs [%s]", err.Error())
		return
	}

	// Store public key string in common
	miner.PubKeyStr, err = shared.EncodePubKey(miner.PrivKey.PublicKey)
	if err != nil {
		miner.Log.Debug("Couldn't encode key: [%s]", err.Error())
	}

	return
}
