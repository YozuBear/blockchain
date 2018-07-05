/*
 * Protocols for connecting to and sending messages between connected miners
 *
 */
package miner

import (
	"../shared"
	"net/rpc"
)

type MinerMinerRPC int

// Called by new miner to be connected
func (mm MinerMinerRPC) Connect(newMiner shared.ConnectMinerArgs, myKey *string) (err error) {
	*myKey, _ = shared.EncodePubKey(PrivKey.PublicKey)
	conn, err := rpc.Dial("tcp", newMiner.IPPort)
	if err != nil {
		return
	}
	UpdateNeighbours(newMiner.EncodedPubKey, conn)
	return nil
}

// Retrieve a block with a given hash
func (mm MinerMinerRPC) GetBlockFromHash(hash string, block *Block) (err error) {
	Log.Debug("A miner is trying to retrieve block wtih hash [%s]", hash)
	*block , err = GetBlock(hash)
	return
}

// RPC call to periodically check up on miners and see if they are still active
func (mm MinerMinerRPC) IsAlive(arg int, reply *bool) (err error) {
	*reply = true
	return
}

func RemoveInactiveMiner(key string) (err error) {
	NeighbourLock.Lock()
	defer NeighbourLock.Unlock()

	conn, _ := ActiveNeighbours[key]
	if conn != nil {
		conn.Close()
	}
	delete(ActiveNeighbours, key)
	NumNeighbours--
	Log.Debug("Inactive miner removed\n Total connections [%v]", NumNeighbours)
	return
}

func UpdateNeighbours(key string, conn *rpc.Client) (err error) {
	NeighbourLock.Lock()
	defer NeighbourLock.Unlock()

	if _, exists := ActiveNeighbours[key]; exists {
		// if neighbour already exists, keep old connection and close the new one
		conn.Close()
		return
	}
	ActiveNeighbours[key] = conn
	NumNeighbours++
	Log.Debug("New neighbour -- Total connections [%v]", NumNeighbours)
	return
}

// ---------------------------------------------------------------------
// Miner Receiving from miner network
// ---------------------------------------------------------------------

// Receive Op from other miners
func (mm MinerMinerRPC) FloodOp(args Op, reply *int) (err error) {
	DisseminateOp(args)
	return nil
}

// Receive OpBlock from other miners
func (mm MinerMinerRPC) FloodOpBlock(args OpBlock, reply *int) (err error) {
	DisseminateBlock(args)
	return nil
}

// Receive NoOpBlock from other miners
func (mm MinerMinerRPC) FloodNoOpBlock(args NoOpBlock, reply *int) (err error) {
	DisseminateBlock(args)
	return nil
}

// When a new miner joins the network, this provides the new miner latest block chain data
func (mm MinerMinerRPC) GetChain(args int, reply *RawGenBlockchain) (err error) {
	rawChain := GetLongestChain()

	processedChain := RawGenBlockchain{}

	for _, block := range rawChain {
		genBlock := GeneralBlock{}
		switch block.(type) {
		case OpBlock:
			genBlock = GeneralBlock{
				PrevHash:    block.(OpBlock).PrevHash,
				PubKeyMiner: block.(OpBlock).PubKeyMiner,
				Nonce:       block.(OpBlock).Nonce,
				Ops:         block.(OpBlock).Ops,
			}
		case NoOpBlock:
			genBlock = GeneralBlock{
				PrevHash:    block.(NoOpBlock).PrevHash,
				PubKeyMiner: block.(NoOpBlock).PubKeyMiner,
				Nonce:       block.(NoOpBlock).Nonce,
				Ops:         []Op{},
			}
		}

		processedChain = append(processedChain, genBlock)
	}

	*reply = processedChain
	return nil
}

// ---------------------------------------------------------------------
// Miner Sending to miner network
// ---------------------------------------------------------------------

// Flood miner network with Block
func FloodMinerNetworkBlock(args Block) (err error) {
	var reply int
	for _, conn := range ActiveNeighbours {
		switch args.(type) {
		case OpBlock:
			err = conn.Call("MinerMinerRPC.FloodOpBlock", args.(OpBlock), &reply)
		case NoOpBlock:
			err = conn.Call("MinerMinerRPC.FloodNoOpBlock", args.(NoOpBlock), &reply)
		default:
			Log.Error("bad type")
		}

		if err != nil {
			Log.Error("rpc call err [%s]", err.Error())
		}
	}
	return nil
}

// Flood miner network with Op
func FloodMinerNetworkOp(args Op) (err error) {
	var reply int
	for _, conn := range ActiveNeighbours {
		err := conn.Call("MinerMinerRPC.FloodOp", args, &reply)
		if err != nil {
			Log.Error("rpc call err [%s]", err.Error())
		}
	}
	return nil
}

// Return the valid, longest chain from neighbours
// Each neighbour returns a raw block chain, need to compare
// and produce our own longest, valid raw block chain.
func GetNetworkBlockchain() RawBlockchain {
	// Obtain raw block chain from each neighbour, store them in blockchains
	var blockchains []RawBlockchain
	for pubHash, conn := range ActiveNeighbours {
		var reply RawGenBlockchain
		var args int
		err := conn.Call("MinerMinerRPC.GetChain", args, &reply)
		if err != nil {
			Log.Error("Unable to make an RPC call to miner [%s], err: %s ", pubHash, err.Error())
		} else {
			Log.Debug("Got a raw blockchain of length [%d] from neighbour", len(reply))

			// convert genblocks in chain to blocks
			rawChain := RawBlockchain{}
			for _, genBlock := range reply {
				if len(genBlock.Ops) == 0 {
					noOpBlock := NoOpBlock{
						PrevHash:    genBlock.PrevHash,
						PubKeyMiner: genBlock.PubKeyMiner,
						Nonce:       genBlock.Nonce,
					}
					rawChain = append(rawChain, noOpBlock)
				} else {
					opBlock := OpBlock{
						PrevHash:    genBlock.PrevHash,
						Ops:         genBlock.Ops,
						PubKeyMiner: genBlock.PubKeyMiner,
						Nonce:       genBlock.Nonce,
					}
					rawChain = append(rawChain, opBlock)
				}
			}

			// add chain to chain array
			blockchains = append(blockchains, rawChain)
		}

	}

	return GetLongestChainInNetwork(blockchains)
}

// Returns the longest chain agreed by the majority of the blockchain copies
func GetLongestChainInNetwork(chains []RawBlockchain) (longestChain RawBlockchain) {

	/* Some local helpers */

	type BlockHashCountTable map[string][]*RawBlockchain // hash -> chains with hash

	getHashWithMaxCount := func(tbl BlockHashCountTable) (maxCount int, maxHash string) {
		for hash, chains := range tbl {
			if len(chains) > maxCount {
				maxCount = len(chains)
				maxHash = hash
			}
		}
		return
	}

	GethMaxLengthOfChains := func(chains []RawBlockchain) int {
		maxLength := 0
		for _, chain := range chains {
			if len(chain) > maxLength {
				maxLength = len(chain)
			}
		}
		return maxLength
	}

	/* Implementation */

	majority := len(chains) / 2

	var maxCount int                          // max number of miners having the same block at the same index
	var maxHash string                        // the hash agreed by the most miners at an index
	var blockHashCountTbl BlockHashCountTable // hash -> chains with hash at an index

	// Start from the highest index
	// look for a block with majority at the index
	// if no such block then decrease index and repeat
	for i := GethMaxLengthOfChains(chains); i >= 0; i-- {
		blockHashCountTbl = make(BlockHashCountTable)

		// get the counts of owners for each block hash existing at the current index
		for _, chain := range chains {
			if len(chain) > i {
				h := chain[i].Hash()
				blockHashCountTbl[h] = append(blockHashCountTbl[h], &chain)
			}
		}

		maxCount, maxHash = getHashWithMaxCount(blockHashCountTbl)

		if maxCount > majority {
			break
		}

		maxHash = ""
	}

	if maxHash == "" {
		// returns empty chain if no majority found
		longestChain = RawBlockchain{}
	} else {
		// get one of the chains containing the latest block with the majority
		longestChain = *(blockHashCountTbl[maxHash][0])
	}

	return
}
