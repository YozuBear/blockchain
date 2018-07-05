/*

This package specifies the application's interface to the the BlockArt
library (blockartlib) to be used in project 1 of UBC CS 416 2017W2.

*/

package blockartlib

import (
	"../shared"
	"crypto/ecdsa"
	"crypto/rand"
	"net/rpc"
)

/* Aliases for shared constants and types */
type CanvasSettings shared.CanvasSettings
type ShapeType shared.ShapeType

const (
	PATH = ShapeType(shared.PATH)
	CIRC = ShapeType(shared.CIRC)
)

var log *shared.Logger

////////////////////////////////////////////////////////////////////////////////////////////
// <ERROR DEFINITIONS>

// These type definitions allow the application to explicitly check
// for the kind of error that occurred. Each API call below lists the
// errors that it is allowed to raise.
//
// Also see:
// https://blog.golang.org/error-handling-and-go
// https://blog.golang.org/errors-are-values

// Contains address IP:port that art node cannot connect to.
// TODO Error method doesn't carry over to alias
type DisconnectedError shared.DisconnectedError
type InsufficientInkError shared.InsufficientInkError
type InvalidShapeSvgStringError shared.InvalidShapeSvgStringError
type ShapeSvgStringTooLongError shared.ShapeSvgStringTooLongError
type InvalidShapeHashError shared.InvalidShapeHashError
type ShapeOwnerError shared.ShapeOwnerError
type OutOfBoundsError shared.OutOfBoundsError
type ShapeOverlapError shared.ShapeOverlapError
type InvalidBlockHashError shared.InvalidBlockHashError

// </ERROR DEFINITIONS>
////////////////////////////////////////////////////////////////////////////////////////////

// Represents a canvas in the system.
type Canvas interface {
	// Adds a new shape to the canvas.
	// Can return the following errors:
	// - DisconnectedError
	// - InsufficientInkError
	// - InvalidShapeSvgStringError
	// - ShapeSvgStringTooLongError
	// - ShapeOverlapError
	// - OutOfBoundsError
	AddShape(validateNum uint8, shapeType ShapeType, shapeSvgString string, fill string, stroke string) (shapeHash string, blockHash string, inkRemaining uint32, err error)

	// Returns the encoding of the shape as an svg string.
	// Can return the following errors:
	// - DisconnectedError
	// - InvalidShapeHashError
	GetSvgString(shapeHash string) (svgString string, err error)

	// Returns the amount of ink currently available.
	// Can return the following errors:
	// - DisconnectedError
	GetInk() (inkRemaining uint32, err error)

	// Removes a shape from the canvas.
	// Can return the following errors:
	// - DisconnectedError
	// - ShapeOwnerError
	DeleteShape(validateNum uint8, shapeHash string) (inkRemaining uint32, err error)

	// Retrieves hashes contained by a specific block.
	// Can return the following errors:
	// - DisconnectedError
	// - InvalidBlockHashError
	GetShapes(blockHash string) (shapeHashes []string, err error)

	// Returns the block hash of the genesis block.
	// Can return the following errors:
	// - DisconnectedError
	GetGenesisBlock() (blockHash string, err error)

	// Retrieves the children blocks of the block identified by blockHash.
	// Can return the following errors:
	// - DisconnectedError
	// - InvalidBlockHashError
	GetChildren(blockHash string) (blockHashes []string, err error)

	// Closes the canvas/connection to the BlockArt network.
	// - DisconnectedError
	CloseCanvas() (inkRemaining uint32, err error)
}

// The constructor for a new Canvas object instance. Takes the miner's
// IP:port address string and a public-private key pair (ecdsa private
// key type contains the public key). Returns a Canvas instance that
// can be used for all future interactions with blockartlib.
//
// The returned Canvas instance is a singleton: an application is
// expected to interact with just one Canvas instance at a time.
//
// Can return the following errors:
// - DisconnectedError
func OpenCanvas(minerAddr string, privKey ecdsa.PrivateKey) (canvas Canvas, setting CanvasSettings, err error) {
	var client *rpc.Client

	client, err = rpc.Dial("tcp", minerAddr)
	if err != nil {
		log.Error("OpenCanvas failed %s", err.Error())
		err = maskRPCServerErr(err, minerAddr)
		return
	}

	h := shared.HashByteArr([]byte(shared.ArtnodeRegMsg))
	r, s, err := ecdsa.Sign(rand.Reader, &privKey, h)

	if err != nil {
		log.Error("OpenCanvas signature failed [%s]", err.Error())
		err = maskRPCServerErr(err, minerAddr)
		return
	}
	err = client.Call("MinerArtRPC.OpenCanvas", &shared.OpenArgs{R: r, S: s}, &setting)

	canvas = &myCanvas{client: client, cSettings: setting, serverAddr: minerAddr}
	return
}

func init() {
	log = shared.NewLogger(false, false, true)
}
