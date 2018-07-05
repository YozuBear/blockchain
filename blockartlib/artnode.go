package blockartlib

import (
	"../shared"
	"net/rpc"
	"strings"
)

type myCanvas struct {
	client     *rpc.Client
	cSettings  CanvasSettings
	serverAddr string
}

/* Implements Canvas interface */
func (c *myCanvas) AddShape(validateNum uint8, shapeType ShapeType, shapeSvgString string, fill string, stroke string) (shapeHash string, blockHash string, inkRemaining uint32, err error) {
	if fill == "" || stroke == "" {
		err = shared.InvalidShapeSvgStringError(shapeSvgString)
		return
	}

	var reply shared.AddReply

	err = c.client.Call(
		"MinerArtRPC.AddShape",
		shared.AddArgs{
			ValidateNum:    validateNum,
			ShapeType:      shared.ShapeType(shapeType),
			ShapeSvgString: shapeSvgString,
			Fill:           fill,
			Stroke:         stroke},
		&reply)

	if err != nil {
		err = maskRPCServerErr(err, c.serverAddr)
		return
	}

	shapeHash = reply.ShapeHash
	blockHash = reply.BlockHash
	inkRemaining = reply.InkRemaining
	return
}

func (c *myCanvas) GetSvgString(shapeHash string) (svgString string, err error) {
	var reply shared.GetReply

	err = c.client.Call(
		"MinerArtRPC.Get",
		shared.GetArgs{
			Type: shared.SVG,
			Hash: shapeHash},
		&reply)

	if err != nil {
		err = maskRPCServerErr(err, c.serverAddr)
		return
	}

	svgString = csvToSvg(reply.Str)
	return
}

func (c *myCanvas) GetInk() (inkRemaining uint32, err error) {
	var reply shared.GetReply

	err = c.client.Call(
		"MinerArtRPC.Get",
		shared.GetArgs{
			Type: shared.INK,
			Hash: c.serverAddr}, // hash will be ignored by the callee
		&reply)

	if err != nil {
		err = maskRPCServerErr(err, c.serverAddr)
		return
	}

	inkRemaining = reply.InkRemaining
	return
}

func (c *myCanvas) DeleteShape(validateNum uint8, shapeHash string) (inkRemaining uint32, err error) {
	var reply shared.RmReply

	err = c.client.Call(
		"MinerArtRPC.RmShape",
		shared.RmArgs{
			ValidateNum: validateNum,
			ShapeHash:   shapeHash},
		&reply)

	if err != nil {
		err = maskRPCServerErr(err, c.serverAddr)
		return
	}

	inkRemaining = reply.InkRemaining
	return
}

func (c *myCanvas) GetShapes(blockHash string) (shapeHashes []string, err error) {
	var reply shared.GetReply

	err = c.client.Call(
		"MinerArtRPC.Get",
		shared.GetArgs{
			Type: shared.SHAPES,
			Hash: blockHash},
		&reply)

	if err != nil {
		err = maskRPCServerErr(err, c.serverAddr)
		return
	}

	shapeHashes = reply.StrArr
	return
}

func (c *myCanvas) GetGenesisBlock() (blockHash string, err error) {
	var reply shared.GetReply

	err = c.client.Call(
		"MinerArtRPC.Get",
		shared.GetArgs{
			Type: shared.GEN,
			Hash: ""}, // hash should be ignored by callee
		&reply)

	if err != nil {
		err = maskRPCServerErr(err, c.serverAddr)
		return
	}

	blockHash = reply.Str
	return
}

func (c *myCanvas) GetChildren(blockHash string) (blockHashes []string, err error) {
	var reply shared.GetReply

	err = c.client.Call(
		"MinerArtRPC.Get",
		shared.GetArgs{
			Type: shared.CHILDREN,
			Hash: blockHash},
		&reply)

	if err != nil {
		err = maskRPCServerErr(err, c.serverAddr)
		return
	}

	blockHashes = reply.StrArr
	return
}

func (c *myCanvas) CloseCanvas() (inkRemaining uint32, err error) {
	err = c.client.Call(
		"MinerArtRPC.CloseCanvas",
		0,
		&inkRemaining)
	if err != nil {
		log.Error("Error closing canvas %s", err.Error())
		err = maskRPCServerErr(err, c.serverAddr)
	}

	c.client.Close()
	return
}

// Converts a comma-separated list of the format:
// [path commmand],[fill],[stroke]
// into an SVG string so that it looks like:
// <path d="[path command]" fill="[fill]" stroke="[stroke]"></path>
func csvToSvg(s string) (svgString string) {
	log.Trace(s)
	svgComponents := strings.Split(s, ",")
	if len(svgComponents) != 4 {
		log.Error("GetSVG returned [%s]", s)
	}

	stype := svgComponents[0]
	svg := svgComponents[1]
	fill := svgComponents[2]
	stroke := svgComponents[3]

	switch stype {
	case "PATH":
		svgString = "<path d=\"" + svg + "\" fill=\"" + fill + "\" stroke=\"" + stroke + "\"></path>"
	case "CIRC":
		circParts := strings.Split(svg, " ")
		svg = ""
		for i := 0; i < len(circParts); i += 2 {
			switch circParts[i] {
			case "cx":
				svg += "cx=\"" + circParts[i+1] + "\" "
			case "cy":
				svg += "cy=\"" + circParts[i+1] + "\" "
			case "r":
				svg += "r=\"" + circParts[i+1] + "\" "
			}
		}
		svgString = "<circle " + svg + "fill=\"" + fill + "\" stroke=\"" + stroke + "\" />"
	default:
		log.Error("Bad ShapeType")
	}

	return

}

func maskRPCServerErr(err error, msg string) (e error) {
	if err == nil {
		return nil
	}

	switch err.(type) {
	case rpc.ServerError:
		e = err
	default:
		e = shared.DisconnectedError(msg)
	}
	return
}
