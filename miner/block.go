package miner

import (
	"../shared"
	"encoding/binary"
	"encoding/hex"
)

// The Block Unit in blockchain
type Block interface {
	// Get the previous block's hash
	GetPrevHash() (hash string)

	// Get the public key of the miner who found the nonce and added the block
	GetMinerPubKey() (minerPubKey string)

	// Get the nonce of the block, produced by the miner indicated by MinerPubKey
	GetNonce() (nonce uint32)

	// Get the operations in OpBlock.
	// ops is nil in NoOpBlock
	GetOps() (ops []Op)

	// Hashes all of the block's fields into string
	Hash() (hashedBlock string)

	GetPowDifficulty() (difficulty uint8)
}

// Op: contains the operation of art note
type Op struct {
	Add      bool            // true if this is an add operation, false if it's a remove operation
	Op       Shape           // Shape drawn by artnode
	OpSig    shared.OpenArgs // Shape operation signed with a private key of miner generator
	PubKey   string          // Public key of the miner owner of this Op
	ValidNum uint8           // Number of blocks required after this block to be rewarded inks
}

// Op Block that implements Block
type OpBlock struct {
	PrevHash    string
	Ops         []Op   // List of operations
	PubKeyMiner string // Public key of the miner who made the Op block
	Nonce       uint32
}

// No-Op block that implements Block
type NoOpBlock struct {
	PrevHash    string
	PubKeyMiner string
	Nonce       uint32
}

// General block struct for holding either NoOp or Op blocks
type GeneralBlock struct {
	PrevHash    string
	PubKeyMiner string
	Nonce       uint32
	Ops         []Op
}

// ---------------------------------------------------------------------
// Hash functions
// ---------------------------------------------------------------------

// Hash Op's fields into byte arrays
// Fields: op, op-signature, pub-key,
func (op Op) HashToBytes() []byte {
	opBytes := op.Op.HashToBytes()
	opSigRBytes, _ := op.OpSig.R.GobEncode()
	opSigSBytes, _ := op.OpSig.S.GobEncode()
	pubKeyBytes := []byte(op.PubKey)
	validNumBytes := make([]byte, 1)
	validNumBytes[0] = byte(op.ValidNum)
	AddBytes := make([]byte, 1)
	if op.Add {
		AddBytes[0] = 1
	} else {
		AddBytes[0] = 0
	}
	args := [5][]byte{opBytes, opSigRBytes, opSigSBytes, pubKeyBytes, validNumBytes}

	opBytes = shared.ConcateByteArr(args[:])
	if !op.Add {
		// Due to weird hashing behaviour that op is hashed to same bytes regardless of Add
		// manually set the bytes differently
		opBytes[0] = 15
	}

	return shared.HashByteArr(opBytes)
}

// Hash Op's fields into string
// Fields: op, op-signature, pub-key,
func (op Op) HashToString() string {
	bytes := op.HashToBytes()
	return hex.EncodeToString(bytes[:])
}

// Hashes OpBlock to string
// Precondition: must contain an ops
func (block OpBlock) Hash() (hashedBlock string) {
	// Precondition check: OpBlock contains ops
	ops := block.GetOps()
	if len(ops) == 0 {
		Log.Error("OpBlock must contain operations")
	}

	// Previous hash bytes
	prevHashBytes := []byte(block.GetPrevHash())

	// Append all the ops within the block
	var opsBytes []byte
	for _, op := range ops {
		opHash := op.HashToBytes()
		opsBytes = append(opsBytes, opHash[:]...)
	}

	// Miner pubkey bytes
	minerPubKeyBytes := []byte(block.GetMinerPubKey())

	// Nonce bytes
	nonceBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(nonceBytes, block.GetNonce())

	args := [4][]byte{prevHashBytes, opsBytes, minerPubKeyBytes, nonceBytes}
	hashedByteArr := shared.HashByteArr(shared.ConcateByteArr(args[:]))

	return hex.EncodeToString(hashedByteArr[:])
}

// Hashes NoOpBlock to string
func (block NoOpBlock) Hash() (hashedBlock string) {
	prevHashBytes := []byte(block.GetPrevHash())
	minerPubKeyBytes := []byte(block.GetMinerPubKey())
	nonceBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(nonceBytes, block.GetNonce())
	args := [3][]byte{prevHashBytes, minerPubKeyBytes, nonceBytes}
	hashedByteArr := shared.HashByteArr(shared.ConcateByteArr(args[:]))

	return hex.EncodeToString(hashedByteArr[:])
}

// Return a new Op with Add field set
func SetOpAdd(op Op, add bool) (newOp Op) {
	newOp = Op{
		Op:       op.Op,
		OpSig:    op.OpSig,
		PubKey:   op.PubKey,
		ValidNum: op.ValidNum,
		Add:      add,
	}

	return newOp
}

// ---------------------------------------------------------------------
// Getters for OpBlock
// ---------------------------------------------------------------------
func (opBlock OpBlock) GetPrevHash() (hash string) {
	return opBlock.PrevHash
}
func (opBlock OpBlock) GetMinerPubKey() (minerPubKey string) {
	return opBlock.PubKeyMiner
}
func (opBlock OpBlock) GetNonce() (nonce uint32) {
	return opBlock.Nonce
}
func (opBlock OpBlock) GetOps() (ops []Op) {
	return opBlock.Ops
}

func (opBlock OpBlock) GetPowDifficulty() (difficulty uint8) {
	return NetSettings.PoWDifficultyOpBlock
}

// ---------------------------------------------------------------------
// Getters for NoOpBlock
// ---------------------------------------------------------------------
func (noOpBlock NoOpBlock) GetPrevHash() (hash string) {
	return noOpBlock.PrevHash
}
func (noOpBlock NoOpBlock) GetMinerPubKey() (minerPubKey string) {
	return noOpBlock.PubKeyMiner
}
func (noOpBlock NoOpBlock) GetNonce() (nonce uint32) {
	return noOpBlock.Nonce
}

// Noop does not have operations
func (noOpBlock NoOpBlock) GetOps() (ops []Op) {
	return ops
}

func (noOpBlock NoOpBlock) GetPowDifficulty() (difficulty uint8) {
	return NetSettings.PoWDifficultyNoOpBlock
}
