# Blockartlib

This was completed by a team of 4, I worked on the main miner blockchain miner logic while my teammates worked on the geometric intersection, UI, and art nodes logic.

[Full project description](https://www.cs.ubc.ca/~bestchai/teaching/cs416_2017w2/project1/index.html)

## Miner-Artnode API

This is the most straightforward API. We have four different RPC calls:
* `OpenCanvas`: opens the canvas and gets canvas settings back
* `CloseCanvas`: closes the canvas
* `AddShape`: Adds a shape
* `RmShape`: Removes a shape
* `Get`: Get a specified value

## Miner-Miner API
* `Connect`: Dials the specified miner and stores the connection
* `GetBlockFromHash`: Retrieves the hash of a given block
* `IsAlive`: Heartbeat between miners
* `FloodOp`: Receive op and disseminate op
* `FloodOpBlock`: Receive op block and disseminate op block
* `FloodNoOpBlock`: Receive noOp block and disseminate noOp block
* `GetChain`: Get the longest chain for new miner

## SVG & Shapes

### SVG Parsing

Each SVG string (not including stroke and fill) is parsed into a list of Component objects. Component is an interface implemented by Point, Group, and Circle:

```
type Component interface {
    ...
}

// Point stores the x and y coordinates of the point
type Point struct {
	X, Y float64
}

// Group is a collection of CONNECTED segments
type Group struct {
	Segments []Segment
}

// Circle stores the radius, and the coordinate of the centre
type Circle struct {
	R, X, Y float64
}

// Segment does NOT implement Component, but is used to
// represent the line segments stored in Group. Each segment
// is a line with a starting point and ending point.
type Segment struct {
	PStart, PEnd Point
	YInt, Slope  float64
}
```

An SVG string is parsed differently according to the shape type. For the CIRC shape type, the SVG parser is expecting a string representing one single circle, e.g. "cx 3 cy 4 r 2" is a circle of radius 2 centered at (3, 4). Parsing is done simply by splitting the string by " " into tokens and checking every other token.  For the PATH shape type, the SVG parser converts the SVG to a point map, which stores points as keys and the key's neighbours as values. The parse then calls a helper to extract single points (i.e. points without neighbours) to get a list of disconnected points, then another helper to extract the groups of connected segments, then join the groups with the points to get a collection of components.

### Shape Validation

Once the parser extracts a collection of components, it is stored in a Shape object, along with the fill, stroke, and the owner's public key string hash: 

```
type Shape struct {
	Owner             string // public key hash of the shape owner
	ShapeType         shared.ShapeType
	Svg, Fill, Stroke string // addshape args
	Components        []Component
}
```

To validate the shape, we first check if the shape is within the canvas. For Point, this is trivial and we simply check if the point is within the boundary; for Group, we check that the end points of each Segment is wihtin the boundary; and for circle, we add radius to the centre to the "max" point and subtract radius from the centre to get the "min" point and check if these points are in the boundary.

Then we check if the fill is valid. A Point cannot have fill, so if fill is not transparent it is invalid. For Group, we check if the segments form a closed loop by comparing the starting point of the first segment and the ending point of the last segment. The Group is a closed shape iff these two points are the same because when we parse it we make sure we trace the neighbours until we reach a point without a neighbour that has not been added. Circle by definition is closed so has valid fill.

Finally we check if the shape overlaps with another shape not owned by the same owner by comparing the components in the shape. We do this by comparing the components in the shape we are adding with the components of every other shape on the canvas. 1) We see if they intersect each other on the border. if not, 2) We check if the new component contains the old component. If not, 3) We check if the old component contains the new component.

#### Step 1
For 1), there are six possible comparisons:

##### Point-Point
This is trivial - we simply compare the coordinates of the two points

##### Point-Group
For each Segment in Group, we check if Point is on the line the contains the Segment and if so, if the Point is between the Segment's endpoints.

##### Point-Circle
We check if the point is on the circumference of the circle.

##### Group-Group
We check if each segment in the groups intersect within the starting points and the ending points.

##### Group-Circle
We use the quadratic equation to solve for intersection point(s) between the circle and each segment and see if the intersection point(s) occur between the segment's start point and end point.

##### Circle-Circle
We get the distance between 2 circles and compare it to the radii to see if the circles touch, are separate, or one is in the other.

#### Steps 2 & 3
For 2) and 3), if fill is transparent, then we know one can't contain the other. If fill is not transparent, then we get a point from the component that we are testing if is contained in the other, find another point OUTSIDE of the filled shape (we use (-1, -1)), and create a test segment between these points. We then count how many times this test segment intersects with the component, and if they intersect an odd number of times, then we know one contains the other, else the other shape is outside or is partly inside (already checked by step 1) 

### Shape Area

##### Point
A point has an area of 0.

##### Group

###### Stroke
We calculate the length (Euclidean norm) of each segment and sum the results up.

###### Fill
We use Green's Theorem to calculate the area of a closed curve. Green stated that the line integral of a function f over a closed curve C is equal to the integral of curl f over the area bounded by C in the counter clockwise direction. Since we want to calculate the area, we know curl f must be equal to 1, and we use that to find f and compute the integral into abs((1/2)Sum(x[i]*y[i+1] - y[i]*x[i+1])).

##### Circle

###### Stroke
We use the circle circumference (2rpi) for the number of units the stroke requires

###### Fill
We use the area of the circle pi(r^2)

## BlockChain  
Adding shape and delete shape to blockchain: stalls until validNum of blocks are attached.  
If ops are added to a side-chain, timer will time-out and checks if the ops are in the log of the longest change.
If ops are not on the longest chain at timeout, the ops are re-added.

##### Flooding:  
When getting an op or block from neighbour, it checks if they're in the log already.
If they are not in the log, they're disseminated to the miner's neighbours.
This prevents infinite loop.

##### Timeout: is adjusted according to validNum for adding shape and deleting shape.  
It's adjusted according to proof of work difficulty for finding nonce, in case more ops are queued, and miner is wasting too much time on mining the out-dated block.

##### Mining: miner mines NoOpBlock if there is no operation in the Op-Queue (which accumulates ops from addShape and neighbour's disseminating ops), otherwise it adds all operations in Op-Queue into one OpBlock.

##### Validations:
###### Block validations:  
- Check that the nonce for the block is valid: PoW is correct and has the right difficulty.  
- Check that each operation in the block has a valid signature (this signature should be generated using the private key and the operation).  
- Check that the previous block hash points to a legal, previously generated, block.
###### Operation validations:  
- Check that each operation has sufficient ink associated with the public key that generated the operation.
- Check that each operation does not violate the shape intersection policy described above.
- Check that the operation with an identical signature has not been previously added to the longest chain in the blockchain. This prevents operation replay attacks.
- Check that an operation that deletes a shape refers to a shape that exists and which has not been previously deleted. 

```
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
	Shape    Shape
	ValidNum uint8
	Add      bool
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
```
