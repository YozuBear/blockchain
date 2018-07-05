package miner

import (
	"../shared"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"strconv"
	"time"
)

// Key: miner's public key
// Val: miner's ink
type InkTable map[string]uint32

// opHash -> shape
type Shapes map[string]*MinerCanvas
type MinerCanvas struct {
	Shape     Shape
	BlockHash string
}

// Shape waiting for validNum == 0 to be added to Canvas
type QueueShape struct {
	Shape     Shape
	ValidNum  uint8
	Add       bool
	BlockHash string
}

// Usage: for new neighbour to acquire the longest blockchain
// [Genesis+1Node, ... , Leaf block]
type RawBlockchain []Block

// Usage: for concrete types implementing Block to be
// used for rpc args
type RawGenBlockchain []GeneralBlock

// Blockchain represented in tree structure
type BlockChainNode struct {
	Block    Block    // The block pointer
	Children []string // hash block array of this block's children.
	Height   int      // The height in the tree
	InkTable InkTable // Stores the inks of each miner up to this block.
	Canvas   Shapes   // All shapes on Canvas from Genesis to this block,
	// not including side branches
	// Key: Op hash
	QueueShapes map[string]*QueueShape // All shapes in queue waiting to be validated by other blocks
	// key: shape hash
	// Operation log, storing all operations from Genesis to this block
	// Usage: check no overlapped operations
	OpLog map[string]*Op
}

// ---------------------------------------------------------------------
// Global variables for blockchain
// ---------------------------------------------------------------------
// Hash tree that represents the blockchain data structure
// Key: the Block's hash
// Value: BlockChainNode pointer
var treeTable map[string]*BlockChainNode

// The block key to longest chain's leaf in treeTable
// This is the block the miner extends.
var longestLeafHash string

// Operations queues to be added to block
// Precondition: op is valid
var opQueue map[string]*Op

// Holds the blocks waiting to be disseminated once
// miner is initialized
var blockWaitQ []Block

// Holds all blocks we can't recognize the prevHash of
var noParentBlocks map[string]Block

// ---------------------------------------------------------------------
// ---------------------------------------------------------------------
// ---------------------- Interface for art nodes ----------------------
// ---------------------------------------------------------------------
// ---------------------------------------------------------------------

// Returns the block hash of the genesis block.
// Can return the following errors:
// - DisconnectedError
func GetGenesisBlockHash() (blockHash string, err error) {
	hash := NetSettings.GenesisBlockHash
	if hash == "" {
		err = shared.DisconnectedError("genesis block hash not initialized")
	}
	return hash, err
}

// Retrieves the immediate children blocks of the block identified by blockHash.
// Can return the following errors:
// - InvalidBlockHashError
func GetChildren(blockHash string) (blockHashes []string, err error) {
	if treeBlock, exist := treeTable[blockHash]; exist {
		blockHashes = treeBlock.Children
	} else {
		err = shared.InvalidBlockHashError(blockHash)
	}
	return blockHashes, err
}

// Retrieves the shapes in a block
// Can return the following errors:
// - InvalidBlockHashError
func GetShapes(blockHash string) (shapeHashes []string, err error) {
	bcNode, exists := treeTable[blockHash]
	if !exists {
		err = shared.InvalidBlockHashError(blockHash)
		return
	}

	for _, op := range bcNode.Block.GetOps() {
		shapeHashes = append(shapeHashes, op.HashToString())
	}
	return
}

// Retrieves the SVG string, fill, stroke of a shape in a comma separated list
// Can return the following errors:
// - InvalidShapeHashError
func GetSVGFields(shapeHash string) (svg string, err error) {
	op, exists := treeTable[longestLeafHash].OpLog[shapeHash]
	if !exists {
		err = shared.InvalidShapeHashError(shapeHash)
		return
	}

	// Op exists, check if it's currently on canvas
	minerCanvasPtr, existsOnCanvas := treeTable[longestLeafHash].Canvas[shapeHash]
	if existsOnCanvas {
		return minerCanvasPtr.Shape.GetSVGFields(), nil
	} else {
		// Does not exist on Canvas, this shape has been deleted
		// Return shape with white filled from opLog (not modifying opLog's op)
		shape := op.Op
		shapePtr := &shape
		shapePtr.EraseShape()
		return shapePtr.GetSVGFields(), nil

	}

}

// Get the amount of ink held by the given public key
func GetInk(pubKey string) uint32 {
	ink, exists := treeTable[longestLeafHash].InkTable[pubKey]
	if !exists {
		ink = 0
	}
	return ink
}

// Adds a new shape to the block chain, stall until op is validated
// validateNum: number of blocks needed after this block to validate this action
// shape: shape to be added
// Checks for
// - sufficient ink
// - shape validity
func AddShapeToBlockChain(validateNum uint8, shape Shape) (opHash string, blockHash string, inkRemaining uint32, err error) {
	// Check for ink sufficiency
	minerInk := treeTable[longestLeafHash].InkTable[PubKeyStr]
	shapeInk := ShapeInk(shape)
	if shapeInk > minerInk {
		inkRemaining = minerInk
		err = shared.InsufficientInkError(minerInk)
		return
	}

	// Check shape validity
	validated, err := ValidateShape(shape)
	if !validated {
		return
	}

	// Package given shape to an Op
	r, s, err := ecdsa.Sign(rand.Reader, PrivKey, shape.HashToBytes())
	if err != nil {
		Log.Error("sign failure at add shape", err)
		return
	}
	opSig := shared.OpenArgs{r, s}
	op := Op{
		Op:       shape,
		OpSig:    opSig,
		PubKey:   PubKeyStr,
		ValidNum: validateNum,
		Add:      true,
	}

	return AddOpToBlockChain(validateNum, op)
}

func AddOpToBlockChain(validateNum uint8, op Op) (opHash string, blockHash string, inkRemaining uint32, err error) {
	// Disseminate op to miner network
	err = DisseminateOp(op)
	if err != nil {
		Log.Error("invalid op", err)
		return
	}

	// Sleep until shape is validated or timeout
	tick := time.Tick(time.Duration(validateNum*5) * time.Second)
	timeout := time.After(2 * time.Duration(validateNum) * time.Minute)
	opHash = op.HashToString()
	for {
		select {
		case <-tick:
			// check if shape appears on the canvas of longest chain leaf
			canvas := treeTable[longestLeafHash].Canvas

			// Return if shape is on Canvas
			if canvasPtr, ok := canvas[opHash]; ok {
				inkTable := treeTable[longestLeafHash].InkTable
				inkRemaining = inkTable[PubKeyStr]
				Log.Trace("shape added to canvas:", canvasPtr.Shape.Svg)
				Log.Trace("ink remaining", inkRemaining)

				return opHash, canvasPtr.BlockHash, inkRemaining, nil
			}
		case <-timeout:
			// timeout, re-add the block
			// TODO need to distinguish between the case where
			// - other miners are adding to a separate branch, and the case where
			// - it's just taking a long time to validate
			Log.Debug("timeout, shape[%s], add[%t], validNum[%d]", op.Op.Svg, op.Add, validateNum)

			return AddOpToBlockChain(validateNum, op)
		default:
			Log.Trace("sleep")
			time.Sleep(time.Duration(validateNum*5) * time.Second)
		}
	}
	return
}

// Removes a shape from the block chain. Inks are returned to the miner.
// validateNum: number of blocks needed after this block to validate this action
// shapeHash: hash of shape to be removed
// Can return the following errors:
// - DisconnectedError
func DeleteShapeFromBlockChain(validateNum uint8, opHash string) (inkRemaining uint32, err error) {
	// need to check that shape exists and is removed by its owner
	// Check that shape exists on canvas
	shape, existsOnCanvas := treeTable[longestLeafHash].Canvas[opHash]
	opPtr, existsOnOpLog := treeTable[longestLeafHash].OpLog[opHash]
	inkRemaining = treeTable[longestLeafHash].InkTable[PubKeyStr]

	if existsOnCanvas && existsOnOpLog {
		shapeOwner := shape.Shape.Owner
		if shapeOwner != PubKeyStr {
			err = shared.ShapeOwnerError(shapeOwner)
			return
		}

		// Convert op to Delete Op
		add := false
		op := SetOpAdd(*opPtr, add)

		// Disseminate op to miner network
		err = DisseminateOp(op)
		if err != nil {
			Log.Error("invalid op", err)
			return inkRemaining, err
		}

		// Sleep until shape is validated or timeout
		tick := time.Tick(time.Duration(validateNum*5) * time.Second)
		timeout := time.After(2 * time.Duration(validateNum) * time.Minute)
		for {
			select {
			case <-tick:
				// check if shape appears on the canvas of longest chain leaf
				canvas := treeTable[longestLeafHash].Canvas

				// Return if shape is no longer on Canvas
				if _, ok := canvas[opHash]; !ok {
					inkTable := treeTable[longestLeafHash].InkTable
					inkRemaining = inkTable[PubKeyStr]
					return inkRemaining, nil
				}
			case <-timeout:
				// timeout, re-add the block
				Log.Trace("timeout", shape, validateNum)
				return DeleteShapeFromBlockChain(validateNum, opHash)
			default:
				time.Sleep(time.Duration(validateNum*5) * time.Second)
			}
		}

	} else {
		err = shared.InvalidShapeHashError(opHash)
	}

	return
}

// ---------------------------------------------------------------------
// ---------------------------------------------------------------------
// ------------------------- Blockchain logic --------------------------
// ---------------------------------------------------------------------
// ---------------------------------------------------------------------

// ---------------------------------------------------------------------
// Blockchain initialization
// ---------------------------------------------------------------------
// InitBlockchain is called after NetSetting is set
func InitBlockchain() (error, chan bool) {
	// Initialize global variables
	treeTable = make(map[string]*BlockChainNode)
	longestLeafHash = NetSettings.GenesisBlockHash
	opQueue = make(map[string]*Op)
	done := make(chan bool)
	blockWaitQ = make([]Block, 0)
	noParentBlocks = make(map[string]Block)

	// Add Genesis block to tree table
	GenesisNode := BlockChainNode{
		Block:       NoOpBlock{},
		Children:    []string{},
		Height:      0,
		InkTable:    make(InkTable),
		Canvas:      make(Shapes),
		QueueShapes: make(map[string]*QueueShape),
		OpLog:       make(map[string]*Op),
	}
	treeTable[longestLeafHash] = &GenesisNode

	// Validate each block and add it to treeTable
	longestChain := GetNetworkBlockchain()
	for _, block := range longestChain {
		blockHashStr := block.Hash()
		if _, ok := treeTable[blockHashStr]; !ok {
			// Check Block is valid before disseminating
			validated, err := ValidateBlock(block)
			if !validated || err != nil {
				Log.Error("initialization block error [%s], [%s]", blockHashStr, err.Error())
				return err, done
			}

			// Add to our own block chain
			AddBlockToBlockchain(block)
		}
	}

	// start mining
	go Mine()

	// dequeue some blocks
	go dequeueBlocks()

	// Add blocks temporarily held off while waiting for initialization
	for len(blockWaitQ) > 0 {
		blockWaitQCpy := blockWaitQ
		for _, block := range blockWaitQCpy {
			DisseminateBlockForce(block)
		}

		if len(blockWaitQ) > len(blockWaitQCpy) {
			blockWaitQ = blockWaitQ[len(blockWaitQCpy):]
		}
	}

	initialized = true
	Log.Debug("BLOCKCHAIN INITIALIZED")

	return nil, done

}

// Periodicially ask for blocks we are missing from neighbours
func dequeueBlocks() {
	for hash, block := range noParentBlocks {
		// Check if we can already disseminate a block in the queue
		if _, exists := treeTable[block.GetPrevHash()]; exists {
			DisseminateBlockForce(block)
			delete(noParentBlocks, hash) // remove from queue
		} else {
			// Else call our neighbours to see if they have the block
			parent := new(Block)
			for _, conn := range ActiveNeighbours {
				err := conn.Call("MinerMinerRPC.GetBlockFromHash", block.GetPrevHash(), parent)
				if err == nil && parent != nil {
					Log.Debug("Found block. Trying to disseminate now [%v]", parent)
					DisseminateBlockForce(*parent)
					DisseminateBlockForce(block)
					delete(noParentBlocks, hash)
					break
				}
			}
		}
	}
	time.Sleep(5000 * time.Millisecond)
}

// Return the longest chain, excluding side branches and genesis block
// Array order [Genesis+1Node, ... , leaf]
func GetLongestChain() []Block {
	var chain RawBlockchain
	if initialized {
		var currentBlockHash = longestLeafHash
		GenesisHash, err := GetGenesisBlockHash()
		if err != nil {
			panic("genesis hash not initialized")
		}

		// starting from leaf, working upward to Genesis block (root)
		// Insert element to fron of array
		for currentBlockHash != GenesisHash {
			block := treeTable[currentBlockHash].Block
			blockHolder := make([]Block, 1)
			blockHolder[0] = block
			chain = append(blockHolder, chain...)
			currentBlockHash = block.GetPrevHash()
		}
	}

	return chain
}

// ---------------------------------------------------------------------
// Blockchain Logistics
// ---------------------------------------------------------------------
// Disseminate Op to other miners in the network.
// Check that an operation with an identical signature has not been
// previously added to the blockchain
func DisseminateOp(op Op) (err error) {
	Log.Debug("Disseminate op: [%s], AddShape:[%t], validNum:[%d]", op.Op.Svg, op.Add, op.ValidNum)
	// If op is not in the queue yet
	// prevent neighbour flood-back infinite loop
	// prevents operation replay attacks
	opHashStr := op.HashToString()
	_, existInLog := treeTable[longestLeafHash].OpLog[opHashStr]
	_, existInQueue := opQueue[opHashStr]

	// Disseminate op only if it doesn't already exist in queue or the chain's log
	if !existInLog && !existInQueue {
		// Check Op is valid before disseminating
		validated, err := ValidateOp(op)
		if !validated || err != nil {
			Log.Error("disseminate op [%v] error [%s]", op, err.Error())
			return err
		}

		// add op to opQueue
		opQueue[opHashStr] = &op

		// broadcast to network
		FloodMinerNetworkOp(op)
	} else {
		Log.Debug("Do not re-add Op to mining queue op hash: [%s]", opHashStr)
	}

	return err
}

func DisseminateBlock(block Block) (err error) {
	if !initialized {
		blockWaitQ = append(blockWaitQ, block)
		return
	}
	return disseminateBlock(block)
}

func DisseminateBlockForce(block Block) (err error) {
	return disseminateBlock(block)
}

// Disseminate block that is successfully mined to other miners in the network
// - do not call directly (called by DisseminateBlock(Forced))
func disseminateBlock(block Block) (err error) {
	// Only disseminate block if block is not in the chain yet
	// (prevent neighbour flood-back infinite loop)
	blockHashStr := block.Hash()
	if _, ok := treeTable[blockHashStr]; !ok {

		// Check Block is valid before disseminating
		validated, err := ValidateBlock(block)
		if !validated || err != nil {
			Log.Error("Disseminate block validation error [%s]", err)
			return err
		}

		// Add to our own block chain
		AddBlockToBlockchain(block)

		// broadcast to network
		FloodMinerNetworkBlock(block)
	}

	return err
}

// Precondition: block is valid
// Add block to block chain if block is valid
// If block is successfully added, ancestors of the block's ValidNum in QueueShapes is decremented by one
func AddBlockToBlockchain(block Block) {
	validated, err := ValidateBlock(block)
	if !validated || err != nil {
		Log.Error("Panic: invariant violated, block is not valid [%v]", err)
	}

	blockHash := block.Hash()
	previousBlockHash := block.GetPrevHash()

	// Update previous block to have this block as its child
	treeTable[previousBlockHash].Children = append(treeTable[previousBlockHash].Children, blockHash)
	previousBlock := treeTable[previousBlockHash]

	// Update canvas and queueShapes
	canvas := previousBlock.Canvas
	queueShapes := previousBlock.QueueShapes

	for key, _ := range queueShapes {
		queueShapes[key].ValidNum--
		if queueShapes[key].ValidNum > 0 {
			// Do nothing
		} else {
			// Add/Delete the shape to/from Canvas
			shape := queueShapes[key].Shape
			if queueShapes[key].Add {
				minerCanvas := MinerCanvas{
					Shape:     shape,
					BlockHash: queueShapes[key].BlockHash,
				}
				canvas[key] = &minerCanvas
			} else {
				// Retrieve key for op's add hash from opLog
				deleteOp := treeTable[longestLeafHash].OpLog[key]
				addOp := SetOpAdd(*deleteOp, true)
				delete(canvas, addOp.HashToString())
			}

			// Delete from qeueShape, it's reached valid num
			delete(queueShapes, key)

		}
	}

	opLog := previousBlock.OpLog
	inkTable := previousBlock.InkTable
	minerPubKey := block.GetMinerPubKey()
	if _, ok := inkTable[minerPubKey]; !ok {
		// Add first ink table entry
		inkTable[minerPubKey] = 0
	}
	minerPubKeySuffix := minerPubKey[len(minerPubKey)-10:]
	switch block.(type) {
	case OpBlock:
		// Reward miner of mining a OpBlock
		inkTable[minerPubKey] += NetSettings.InkPerOpBlock

		// Loop through ops to perform chain logistics
		// ink transactions, opLog update, queueShapes updates
		for _, op := range block.GetOps() {
			Log.Debug("Op[%s] in Block[%s] queued for validation", op.HashToString(), block.Hash())

			// add current ops to queue shapes
			qs := QueueShape{
				Shape:     op.Op,
				ValidNum:  op.ValidNum,
				Add:       op.Add,
				BlockHash: block.Hash(),
			}
			queueShapes[op.HashToString()] = &qs

			// Add op to log
			opLog[op.HashToString()] = &op

			// delete them from opQueue
			delete(opQueue, op.HashToString())

			// reflect Ops cost
			cost := ShapeInk(op.Op)
			if op.Add {
				inkTable[op.PubKey] -= cost
			} else {
				inkTable[op.PubKey] += cost
			}
			Log.Debug("Add OpBlock. Ink: [%d], Miner Suffix:[%s]", inkTable[minerPubKey], minerPubKeySuffix)
		}
	case NoOpBlock:
		// Reward miner of mining a NoOpBlock
		inkTable[minerPubKey] += NetSettings.InkPerNoOpBlock
		Log.Debug("Add NoOpBlock. Ink: [%d], Miner Suffix:[%s]", inkTable[minerPubKey], minerPubKeySuffix)
	default:
		// Invariant check
		Log.Error("Panic: this is not a valid block")
	}

	// Package into a block chain tree
	bct := BlockChainNode{
		Block:       block,
		Children:    make([]string, 0),
		Height:      previousBlock.Height + 1,
		InkTable:    inkTable,
		Canvas:      canvas,
		QueueShapes: queueShapes,
		OpLog:       opLog,
	}

	treeTable[blockHash] = &bct

	// update the longestLeafHash
	longestLeafHeight := treeTable[longestLeafHash].Height
	if bct.Height > longestLeafHeight {
		Log.Debug("newHash [%s], oldHash [%s], old len [%d],  new len [%d]",
			block.Hash(), longestLeafHash, longestLeafHeight, bct.Height)

		longestLeafHash = block.Hash()
	} else if bct.Height == longestLeafHeight {
		// Randomly chooses one as the longest leaf
		if shared.RandomNumGenerator()%2 == 0 {
			// Debugging
			Log.Debug("Switching to new branch in blockchain, oldHash[%s], oldChain len [%d], new hash [%s], newChain len [%d]", longestLeafHash, longestLeafHeight, block.Hash(), bct.Height)

			longestLeafHash = block.Hash()

		}
	} else {
		Log.Debug("Adding a new node to non-longest side chain: blockhash [%s], longestChainLen [%s], currChainLength [%d]", blockHash, longestLeafHash, previousBlock.Height+1)
	}

}

// Validate the block received from other miners
// Return true if:
// - the nonce for the block is valid: PoW is correct and has the right difficulty.
// - the previous block hash points to a legal, previously generated, block.
// - each operation in the block is valid
// Return false otherwise. (with reason stated in err)
func ValidateBlock(block Block) (validated bool, err error) {
	// Verify nonce is valid
	validated = ZeroPrefix(block)
	if !validated {
		return false, shared.InvalidBlockHashError("invalid nonce")
	}

	// Check previous block exists in blockchain
	_, previousBlockExists := treeTable[block.GetPrevHash()]
	if !previousBlockExists {
		// Add the block to the queue
		noParentBlocks[block.Hash()] = block
		return false, shared.InvalidBlockHashError("previous block pointer is not in blockchain")
	}

	// Check each operation in block is valid
	ops := block.GetOps()
	for _, op := range ops {
		validated, err = ValidateOp(op)
		if !validated || err != nil {
			return validated, err
		}
	}

	return validated, err
}

// Validate an op received from other miners
// Return true if:
// - the operation has a valid signature
// - the operation has sufficient ink associated with the public key that generated the operation
// - the operation does not violate the shape intersection policy
// - an operation that deletes a shape refers to a shape that exists and which has not been previously deleted.
// Return false otherwise. (with reason stated in err)
func ValidateOp(op Op) (validated bool, err error) {
	// Verify signature
	opPubKey, err := shared.DecodePubKey(op.PubKey)
	if err != nil {
		Log.Error("decode pubkey from str failed", err)
		return false, errors.New("decode pubkey from str failed")
	}
	validated = ecdsa.Verify(opPubKey, op.Op.HashToBytes(), op.OpSig.R, op.OpSig.S)
	if !validated {
		return validated, shared.InvalidShapeHashError("shape hash not signed by provided pub key")
	}

	if op.Add {
		// Verify sufficient ink
		minerInk := treeTable[longestLeafHash].InkTable[op.PubKey]
		opCost := ShapeInk(op.Op)
		if opCost > minerInk {
			return false, shared.InsufficientInkError(minerInk)
		}

		// Verify validity of adding shape
		validated, err = ValidateShape(op.Op)
		if !validated {
			return validated, err
		}

	} else {
		// Verify delete shape exists on canvas
		add := true
		addOp := SetOpAdd(op, add)
		_, shapeExistsOnCanvas := treeTable[longestLeafHash].Canvas[addOp.HashToString()]
		if !shapeExistsOnCanvas {
			return false, shared.ShapeOwnerError("shape doesn't exist on canvas")
		}

		// Verify shape hasn't been previously deleted
		shapeQueue, shapeExistsInQueue := treeTable[longestLeafHash].QueueShapes[op.HashToString()]
		if shapeExistsInQueue {
			if !shapeQueue.Add {
				return false, errors.New("shape was in queue to be deleted")
			}
		}

	}

	return validated, err
}

// Return nil if block doesn't exist
func GetBlock(blockHash string) (block Block, err error) {
	blockNode, exist := treeTable[blockHash]
	if exist {
		block = blockNode.Block
	} else {
		err = shared.InvalidBlockHashError(blockHash)
	}

	return block, err
}

// ---------------------------------------------------------------------
// Shapes bridging calls to miner-art
// ---------------------------------------------------------------------

// Calls shape's validate
func ValidateShape(shape Shape) (validated bool, err error) {
	var shapes = make(map[string]Shape)
	latestShapesOnCanvas := treeTable[longestLeafHash].Canvas

	for key, canvasPtr := range latestShapesOnCanvas {
		shapes[key] = canvasPtr.Shape
	}

	canvas := Canvas{
		Shapes: shapes,
		XMax:   NetSettings.CanvasSettings.CanvasXMax,
		YMax:   NetSettings.CanvasSettings.CanvasYMax,
	}

	Log.Debug("validating shape [%s], canvas size [%d]", shape.Svg, len(canvas.Shapes))
	err = shape.Validate(canvas)

	return err == nil, err
}

// Return the total cost of ink from given shape
func ShapeInk(shape Shape) (cost uint32) {
	area, _ := shape.Area()
	Log.Debug("Area of shape [%d]", area)
	return uint32(area)
}

// ---------------------------------------------------------------------
// Local Mining
// ---------------------------------------------------------------------

// Find the nonce for the given block data
// Output same block with the nonce field set
// If it cannot find nonce within timeout, it returns nil
func FindNonce(block Block) Block {

	var (
		opBlk           OpBlock
		noOpBlk         NoOpBlock
		isOpBlk         bool = false
		timeOutDuration time.Duration
	)

	switch block.(type) {
	case OpBlock:
		opBlk = block.(OpBlock)
		isOpBlk = true
		timeOutDuration = time.Duration(NetSettings.PoWDifficultyOpBlock * 25)
	case NoOpBlock:
		noOpBlk = block.(NoOpBlock)
		timeOutDuration = time.Duration(NetSettings.PoWDifficultyNoOpBlock * 25)
	default:
		Log.Error("unsupported Block type")
	}

	// Set time, once ticked, give up current task and work on new block
	// Prevent wasting time
	timeout := time.After(timeOutDuration * time.Second)

	for {
		select {
		case <-timeout:
			return nil
		default:
			// Generate random nonce
			nonce := shared.RandomNumGenerator()

			if isOpBlk {
				opBlk.Nonce = nonce
				if ZeroPrefix(opBlk) {
					return opBlk
				}
			} else {
				noOpBlk.Nonce = nonce
				if ZeroPrefix(noOpBlk) {
					return noOpBlk
				}
			}
		}

	}
}

// Return true if the block hash's prefix is 0.
// Prefix length: proof of work difficulty of the block
func ZeroPrefix(block Block) bool {
	hashString := block.Hash()
	// Check if prefix has the right number of 0
	prefix := hashString[:block.GetPowDifficulty()]
	prefixInt, _ := strconv.ParseInt(prefix, 16, 0)

	return prefixInt == 0
}

// Mining ink by doing proof of work with the operations in queue
// Mine NoOpBlock if there is no op in queue
func Mine() {
	for {
		// Get all queue ops if any, use opLoad to fix content
		opLoad := opQueue

		// Extending from the longestLeafHash
		previousHash := longestLeafHash
		var block Block
		if len(opLoad) > 0 {
			// Construct an OpBlock
			var ops []Op
			for _, opPtr := range opLoad {
				ops = append(ops, *opPtr)
				Log.Trace("mining OpBlock: ", opPtr.Op.Svg)
			}

			block = OpBlock{
				PrevHash:    previousHash,
				Ops:         ops,
				PubKeyMiner: PubKeyStr,
				Nonce:       0,
			}

		} else {
			// Construct a NoOpBlock
			block = NoOpBlock{
				PrevHash:    previousHash,
				PubKeyMiner: PubKeyStr,
				Nonce:       0,
			}
		}

		block = FindNonce(block)

		// If nonce is found
		if block != nil {
			// Disseminate the block
			DisseminateBlock(block)

		}

	}

}
