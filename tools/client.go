package main

import (
	"../blockartlib"
	"bufio"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Error string

func (e Error) Error() string {
	return fmt.Sprintf("#### Error: [%s]", string(e))
}

var (
	canvas   blockartlib.Canvas
	settings blockartlib.CanvasSettings
)

func main() {
	buf := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		input, err := buf.ReadBytes('\n')

		if err != nil {
			fmt.Println(err)
		} else {
			var e error
			var args []string

			parts := strings.Split(strings.Trim(string(input), "\r\n"), "\"")
			if len(parts) == 3 {
				args = strings.Split(strings.Trim(parts[0], " "), " ")
				args = append(args, strings.Trim(parts[1], " "))
				args = append(args, strings.Split(strings.Trim(parts[2], " "), " ")...)
			} else if len(parts) == 1 {
				args = strings.Split(parts[0], " ")
			} else {
				e = Error("Bad input")
			}

			switch args[0] {
			case "opencanvas":
				e = Open(args[1:])
				break
			case "closecanvas":
				e = Close(args[1:])
				break
			case "addshape":
				e = Add(args[1:])
				break
			case "deleteshape":
				e = Remove(args[1:])
				break
			case "getink":
				e = GetInk(args[1:])
				break
			case "getsvg":
				e = GetSVG(args[1:])
				break
			case "getshapes":
				e = GetShapes(args[1:])
				break
			case "getgenesis":
				e = GetGen(args[1:])
				break
			case "getchildren":
				e = GetChildren(args[1:])
				break
			case "q":
				fmt.Println("#### client exiting")
				os.Exit(0)
			default:
				usage()
			}

			if e != nil {
				fmt.Println(e.Error())
				e = nil
			}
		}
	}
}

func usage() {
	fmt.Printf("type 'h' to see usage:\n - opencanvas <miner IP:Port> <private key>\n - closecanvas\n - addshape <validateNum> PATH|CIRC \"<svg>\" <fill> <stroke>\n - deleteshape <validateNum> <shapeHash>\n - getink\n - getsvg <shapeHash>\n - getshapes <blockHash>\n - getgenesis\n - getchildren <blockHash>\n")
}

func Open(args []string) error {
	if len(args) != 2 {
		return Error("open takes 2 args")
	}

	if canvas != nil {
		return Error("canvas already exists. Call closecanvas")
	}

	minerAddr := args[0]
	privKey, err := getPrivKey(args[1])
	if err != nil {
		return err
	}

	canvas, settings, err = blockartlib.OpenCanvas(minerAddr, *privKey)
	if err != nil {
		Output(err.Error())
	} else {
		Output(fmt.Sprintf("settings: %v", settings))
	}

	return nil
}

func Close(args []string) error {
	if len(args) != 0 {
		return Error("close takes 0 args")
	}

	if canvas == nil {
		return Error("canvas does not exist. Call opencanvas")
	}

	ink, err := canvas.CloseCanvas()
	if err != nil {
		Output(err.Error())
	} else {
		Output(fmt.Sprintf("inkRemaining %d", ink))
		canvas = nil
	}

	return nil
}

func Add(args []string) error {
	if len(args) != 5 {
		return Error("addshapes takes 5 args")
	}

	if canvas == nil {
		return Error("canvas does not exist. Call opencanvas")
	}

	validateNum, err := strconv.ParseUint(args[0], 10, 8)
	shapeType, err := getShape(args[1])
	svg := removeQuotes(args[2])
	fill := args[3]
	stroke := args[4]
	if err != nil {
		return err
	}

	shapeHash, blockHash, ink, err := canvas.AddShape(uint8(validateNum), shapeType, svg, fill, stroke)
	if err != nil {
		Output(err.Error())
	} else {
		Output(fmt.Sprintf("shapeHash [%s], blockHash [%s], ink remaining [%d]", shapeHash, blockHash, ink))
	}

	return nil
}

func Remove(args []string) error {
	if len(args) != 2 {
		return Error("deleteshapes takes 2 args")
	}

	if canvas == nil {
		return Error("canvas does not exist. Call opencanvas")
	}

	validateNum, err := strconv.ParseUint(args[0], 10, 8)
	shapeHash := args[1]

	ink, err := canvas.DeleteShape(uint8(validateNum), shapeHash)
	if err != nil {
		Output(err.Error())
	} else {
		Output(fmt.Sprintf("ink remaining [%d]", ink))
	}

	return nil
}

func GetInk(args []string) error {
	if len(args) != 0 {
		return Error("getink takes 0 args")
	}

	if canvas == nil {
		return Error("canvas does not exist. Call opencanvas")
	}

	ink, err := canvas.GetInk()
	if err != nil {
		Output(err.Error())
	} else {
		Output(fmt.Sprintf("inkRemaining: %d", ink))
	}

	return nil
}

func GetSVG(args []string) error {
	if len(args) != 1 {
		return Error("getsvg takes 1 args")
	}

	if canvas == nil {
		return Error("canvas does not exist. Call opencanvas")
	}

	svg, err := canvas.GetSvgString(args[0])
	if err != nil {
		Output(err.Error())
	} else {
		Output(fmt.Sprintf("svg string: %s", svg))
	}

	return nil
}

func GetShapes(args []string) error {
	if len(args) != 1 {
		return Error("getshapes takes 1 args")
	}

	if canvas == nil {
		return Error("canvas does not exist. Call opencanvas")
	}

	shapeHashes, err := canvas.GetShapes(args[0])
	if err != nil {
		Output(err.Error())
	} else {
		Output(fmt.Sprintf("shape hashes: %v", shapeHashes))
	}

	return nil
}

func GetGen(args []string) error {
	if len(args) != 0 {
		return Error("getgenesis takes 0 args")
	}

	if canvas == nil {
		return Error("canvas does not exist. Call opencanvas")
	}

	blockHash, err := canvas.GetGenesisBlock()
	if err != nil {
		Output(err.Error())
	} else {
		fmt.Println(blockHash)
		Output(fmt.Sprintf("block hash: %s", blockHash))
	}

	return nil
}

func GetChildren(args []string) error {
	if len(args) != 1 {
		return Error("getchildren takes 1 args")
	}

	if canvas == nil {
		return Error("canvas does not exist. Call opencanvas")
	}

	blockHashes, err := canvas.GetChildren(args[0])
	if err != nil {
		Output(err.Error())
	} else {
		Output(fmt.Sprintf("block hashes: %v", blockHashes))
	}

	return nil
}

func Output(msg string) {
	fmt.Println("-->", msg)
}

func getShape(shapeType string) (st blockartlib.ShapeType, err error) {
	switch shapeType {
	case "PATH":
		st = blockartlib.PATH
	case "CIRC":
		st = blockartlib.CIRC
	default:
		err = Error(fmt.Sprintf("bad type %s", shapeType))
	}
	return
}

func removeQuotes(str string) string {
	return strings.Replace(str, "\"", "", -1)
}

func getPrivKey(privKeyStr string) (*ecdsa.PrivateKey, error) {
	var privKey *ecdsa.PrivateKey
	h, err := hex.DecodeString(privKeyStr)
	if err != nil {
		return nil, err
	}

	privKey, err = x509.ParseECPrivateKey(h)
	if err != nil {
		return nil, err
	}

	return privKey, nil
}
