package miner

import (
	"../shared"
	"crypto/ecdsa"
)

type MinerArtRPC int

var (
	artnodeRegHash []byte
	localIPPort    string
)

func (ma MinerArtRPC) OpenCanvas(args shared.OpenArgs, reply *shared.CanvasSettings) (err error) {
	Log.Trace(args)
	Log.Trace(&(PrivKey.PublicKey), artnodeRegHash, args.R, args.S)

	validKey := ecdsa.Verify(&(PrivKey.PublicKey), artnodeRegHash, args.R, args.S)
	if !validKey {
		return shared.DisconnectedError(localIPPort)
	}

	*reply = NetSettings.CanvasSettings

	return
}

func (ma MinerArtRPC) CloseCanvas(args int, reply *uint32) (err error) {
	Log.Trace(args)
	return
}

func (ma MinerArtRPC) AddShape(args shared.AddArgs, reply *shared.AddReply) (err error) {
	Log.Trace(args)

	shape := Shape{
		Owner:     PubKeyStr,
		ShapeType: args.ShapeType,
		Svg:       args.ShapeSvgString,
		Fill:      args.Fill,
		Stroke:    args.Stroke,
	}

	shapeHash, blockHash, inkRemaining, err := AddShapeToBlockChain(args.ValidateNum, shape)
	if err != nil {
		return
	}

	*reply = shared.AddReply{shapeHash, blockHash, inkRemaining}

	return
}

func (ma MinerArtRPC) RmShape(args shared.RmArgs, reply *shared.RmReply) (err error) {
	Log.Trace(args)

	inkRemaining, err := DeleteShapeFromBlockChain(args.ValidateNum, args.ShapeHash)
	if err != nil {
		return
	}

	*reply = shared.RmReply{inkRemaining}

	return
}

func (ma MinerArtRPC) Get(args shared.GetArgs, reply *shared.GetReply) (err error) {
	Log.Trace(args)

	var (
		intReply    uint32
		strReply    string
		strArrReply []string
	)

	switch args.Type {
	case shared.INK:
		intReply = GetInk(PubKeyStr)
	case shared.SVG: // given shapeHash
		strReply, err = GetSVGFields(args.Hash)
	case shared.SHAPES:
		strArrReply, err = GetShapes(args.Hash)
	case shared.GEN:
		strReply, err = GetGenesisBlockHash()
	case shared.CHILDREN:
		strArrReply, err = GetChildren(args.Hash)
	default:
		Log.Error("Invalid Get TYPE %d", args.Type)
	}

	if err != nil {
		return
	}

	*reply = shared.GetReply{intReply, strReply, strArrReply}

	return
}
