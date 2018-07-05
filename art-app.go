/*

A blockart application that collaborates with other art apps to draw
an image on a canvas, then produces an html file with the image created.

Usage:
go run art-app.go [miner ip:port] [private key] [path to file with SVG commands]

Format for SVG commands:
Separate each of the following by new lines:
	For paths: d command, followed by fill and then stroke, each separated by a comma
	For circle: cx, cy, and r attributes separated by spaces, followed by fill and then stroke, each separated by a comma
	To delete a path, "D" followed by a comma, then the d command for that shape
	To delete a circle, "D" followed by a comma, then the cx, cy, and r attributes

Some valid commands:
M 1 1 L 3 3 Z
D,M 1 1 L 3 3 Z
cx 5 cy 5 r 2
D,cx 5 cy 5 r 2
*/

package main

// Expects blockartlib.go to be in the ./blockartlib/ dir, relative to
// this art-app.go file
import (
	"./art-app-tests"
	"./blockartlib"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

var paths []string
var commandToShapeHash map[string]string
var htmlFileName = "art_app_svg.html"
var validateNum = uint8(2)

func main() {
	// testGetLongestChain() // uncomment to test getLongestChain()
	commandToShapeHash = make(map[string]string)

	// get args
	args := os.Args[1:]
	minerAddr := args[0]
	privKeyStr := args[1]
	privKey, err := DecodePrivKey(privKeyStr)
	if checkError(err) != nil {
		return
	}
	paths := parseSvgCommands(args[2])
	for _, s := range paths {
		fmt.Println("[art-app.go]: svg command:", s)
	}

	// open canvas
	var c blockartlib.Canvas
	var s blockartlib.CanvasSettings
	c, s, err = blockartlib.OpenCanvas(minerAddr, *privKey)
	if checkError(err) != nil {
		return
	}

	// create paths if there is sufficient ink
	var ink, _ = c.GetInk()
	for _, path := range paths {
		for ink == 0 {
			fmt.Println("[art-app.go] No ink remaining")
			time.Sleep(1000 * time.Millisecond)
			ink, _ = c.GetInk()
		}
		fmt.Println("[art-app.go] Ink remaining:", ink)

		// see what kind of command we have
		isPath, isDelete := commandType(path)

		var shapeType blockartlib.ShapeType
		if isPath == true {
			shapeType = blockartlib.PATH
		} else {
			shapeType = blockartlib.CIRC
		}

		if isDelete == true { // delete shape
			fmt.Println("[art-app.go] Deleting:", path)
			shapeHash := commandToShapeHash[strings.Split(path, ",")[1]]
			_, err = c.DeleteShape(validateNum, shapeHash) //ignore ink remaining
			if checkError(err) != nil {
				return
			}
		} else { // add shape
			fmt.Println("[art-app.go] Adding:", path)
			path, fill, stroke := splitPath(path)
			shapeHash, _, _, err := c.AddShape(validateNum, shapeType, path, fill, stroke)
			if checkError(err) != nil {
				return
			}
			commandToShapeHash[path] = shapeHash
		}
	}

	// get the canvas image
	genBlockHash, err := c.GetGenesisBlock()
	if checkError(err) != nil {
		return
	}
	fmt.Println("[art-app.go] Genesis block hash:", genBlockHash)
	longestChain, err := getLongestChain(c, genBlockHash, false)
	if checkError(err) != nil {
		return
	}
	fmt.Println("[art-app.go] Longest chain", longestChain)

	var shapeHashes []string
	for _, blockHash := range longestChain {
		moreShapeHashes, err := c.GetShapes(blockHash)
		if checkError(err) != nil {
			return
		}
		shapeHashes = append(shapeHashes, moreShapeHashes...)
	}
	fmt.Println("[art-app.go] Shape hashes:", shapeHashes)

	var svgStrings []string
	for _, hash := range shapeHashes {
		svg, err := c.GetSvgString(hash)
		if checkError(err) != nil {
			return
		}
		svgStrings = append(svgStrings, svg)
	}

	fmt.Println("[art-app.go] SVG strings:", svgStrings)

	// generate html file
	f, err := os.Create(htmlFileName)
	if checkError(err) != nil {
		return
	}
	_, err = f.Write([]byte("<svg viewBox=\"0 0 " + strconv.Itoa(int(s.CanvasXMax)) + " " + strconv.Itoa(int(s.CanvasYMax)) + "\">")) // ignore num bytes written
	if checkError(err) != nil {
		return
	}

	for _, s := range svgStrings {
		_, err = f.Write([]byte(s))
	}
	if checkError(err) != nil {
		return
	}
	_, err = f.Write([]byte("</svg>"))

}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "[art-app.go] Error ", err.Error())
		return err
	}
	return nil
}

func DecodePrivKey(privKeyStr string) (*ecdsa.PrivateKey, error) {
	privKeyHex, err := hex.DecodeString(privKeyStr)
	if err != nil {
		return nil, err
	}

	return x509.ParseECPrivateKey(privKeyHex)
}

type block struct {
	parentBlock *block
	blockHash   string
}

// Returns a list of all the block hashes of the blocks in the longest chain
func getLongestChain(c blockartlib.Canvas, genHash string, testFn bool) ([]string, error) {
	var longestChain []string
	var err error

	// maps depth (in blockchain) -> nodes at that depth
	allBlocks := make(map[int][]block)
	// initialize depth 0 to the genesis
	allBlocks[0] = []block{block{parentBlock: nil, blockHash: genHash}}

	// this loop groups the blocks by depth (ie: distance from genesis node)
	currDepth := 0
	for {
		allBlocks[currDepth+1] = make([]block, 0)
		// fetch all blocks one depth below
		for _, blk := range allBlocks[currDepth] {
			var children []string
			if testFn == false {
				children, err = c.GetChildren(blk.blockHash)
				if err != nil {
					return longestChain, err
				}
			} else {
				children = art_app_tests.GetChildren(blk.blockHash)
			}
			// map the blocks to the depth they belong to
			for _, child := range children {
				parent := blk
				allBlocks[currDepth+1] = append(allBlocks[currDepth+1], block{parentBlock: &parent, blockHash: child})
				fmt.Printf("[art-app.go]Adding child [%v] with parent [%v] (address [%v])to depth [%v]\n", child, blk, &blk, currDepth+1)
			}

		}
		// break if we are at a leaf block
		if len(allBlocks[currDepth+1]) == 0 {
			break
		}
		currDepth++
	}
	// currDepth gives us the depth of the longest chain
	// fetch the first block from that depth (if there is more than one longest chain)
	currBlk := allBlocks[currDepth][0]
	longestChain = append(longestChain, currBlk.blockHash)

	for i := currDepth - 1; i >= 0; i-- { // i is depth of currBlk
		currBlk = *currBlk.parentBlock
		longestChain = append([]string{currBlk.blockHash}, longestChain...)
	}

	fmt.Println("[art-app.go] Genesis:", genHash, "\nRoot node:", currBlk.blockHash)
	return longestChain, err
}

// Call this in the main function to test the trees in ./art-app-test/client-longest-chain.go
func testGetLongestChain() {
	for i := 1; i <= 3; i++ {
		genHash, expectedChain := art_app_tests.GetLongestChainTest(i)
		longestChain, _ := getLongestChain(nil, genHash, true)
		fmt.Println("actual:", longestChain, "\nexpected:", expectedChain)
	}
}

// Return the SVG commands in the file with pathname path as a string array
// Each command is a separate string
func parseSvgCommands(path string) (svgCommands []string) {
	fmt.Println("[art-app.go] Parsing svg commands")
	file, err := os.Open(path)
	if checkError(err) != nil {
		return
	}
	//fmt.Println("[art-app.go] Opened file:", path)

	buffer, err := ioutil.ReadAll(file)
	if checkError(err) != nil {
		return
	}
	fmt.Println("[art-app.go] File contents:[" + string(buffer[:]) + "]")

	svgCommands = strings.Split(string(buffer[:]), "\n")
	return
}

// Tells you if the given command is a path command, a delete command, or both
func commandType(command string) (isPath bool, isDelete bool) {
	isPath = false
	isDelete = false
	if command[0] == []byte("M")[0] {
		isPath = true
	} else if command[0] == []byte("D")[0] {
		isDelete = true
		if command[2] == []byte("M")[0] {
			isPath = true
		}
	}
	return
}

func splitPath(path string) (string, string, string) {
	s := strings.Split(path, ",")
	return s[0], s[1], s[2]
}
