package shared

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"net"
)

/*
 * blockartlib types and constants
 */

const (
	// Path shape.
	PATH ShapeType = iota
	CIRC

	// Circle shape (extra credit).
	// CIRCLE
)

// Settings for a canvas in BlockArt.
type CanvasSettings struct {
	// Canvas dimensions
	CanvasXMax uint32
	CanvasYMax uint32
}

// Represents a type of shape in the BlockArt system.
type ShapeType int

/*
 * Artnode-Miner RPC types and constants
 */

type GetType int

const (
	INK GetType = iota
	SVG
	SHAPES
	GEN
	CHILDREN

	ArtnodeRegMsg = "message to be signed by artnode for key verification"
)

type OpenArgs struct {
	R, S *big.Int // signed hash
}

type AddArgs struct {
	ValidateNum                  uint8
	ShapeType                    ShapeType
	ShapeSvgString, Fill, Stroke string
}

type RmArgs struct {
	ValidateNum uint8
	ShapeHash   string
}

type GetArgs struct {
	Type GetType
	Hash string
}

type AddReply struct {
	ShapeHash    string
	BlockHash    string
	InkRemaining uint32
}

type RmReply struct {
	InkRemaining uint32
}

type GetReply struct {
	InkRemaining uint32   // (GetInk)
	Str          string   // svgString (GetSvgString) or blockHash (GetGenesisBlock)
	StrArr       []string // shapeHashes[] (GetShapes) or blockHash[] (GetChildren)
}

/*
 * Error Definitions
 */

// Contains address IP:port that art node cannot connect to.
type DisconnectedError string

func (e DisconnectedError) Error() string {
	return fmt.Sprintf("BlockArt: cannot connect to [%s]", string(e))
}

// Contains amount of ink remaining.
type InsufficientInkError uint32

func (e InsufficientInkError) Error() string {
	return fmt.Sprintf("BlockArt: Not enough ink to addShape [%d]", uint32(e))
}

// Contains the offending svg string.
type InvalidShapeSvgStringError string

func (e InvalidShapeSvgStringError) Error() string {
	return fmt.Sprintf("BlockArt: Bad shape svg string [%s]", string(e))
}

// Contains the offending svg string.
type ShapeSvgStringTooLongError string

func (e ShapeSvgStringTooLongError) Error() string {
	return fmt.Sprintf("BlockArt: Shape svg string too long [%s]", string(e))
}

// Contains the bad shape hash string.
type InvalidShapeHashError string

func (e InvalidShapeHashError) Error() string {
	return fmt.Sprintf("BlockArt: Invalid shape hash [%s]", string(e))
}

// Contains the bad shape hash string.
type ShapeOwnerError string

func (e ShapeOwnerError) Error() string {
	return fmt.Sprintf("BlockArt: Shape owned by someone else [%s]", string(e))
}

// Empty
type OutOfBoundsError struct{}

func (e OutOfBoundsError) Error() string {
	return fmt.Sprintf("BlockArt: Shape is outside the bounds of the canvas")
}

// Contains the hash of the shape that this shape overlaps with.
type ShapeOverlapError string

func (e ShapeOverlapError) Error() string {
	return fmt.Sprintf("BlockArt: Shape overlaps with a previously added shape [%s]", string(e))
}

// Contains the invalid block hash.
type InvalidBlockHashError string

func (e InvalidBlockHashError) Error() string {
	return fmt.Sprintf("BlockArt: Invalid block hash [%s]", string(e))
}

// Arguments to contact the server
type RegisterArgs struct {
	Address   net.Addr
	Key       ecdsa.PublicKey
	PubKeyStr string
}

// Arguments for Miner-Miner RPC calls
type ConnectMinerArgs struct {
	IPPort        string
	EncodedPubKey string
}
