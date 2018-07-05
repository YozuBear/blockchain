package miner

import (
	"../shared"
	"strconv"
	"strings"
)

type AdjacencyMap map[Point][]Point

const (
	maxSVGLen = 128
)

// Parses SVG string into Comoponents
func ParseSVGString(svg string, shapeType shared.ShapeType) (components []Component, err error) {
	if len(svg) > maxSVGLen {
		return []Component{}, shared.ShapeSvgStringTooLongError(svg)
	}

	switch shapeType {
	case shared.PATH:
		components, err = parseSVGPath(svg)
	case shared.CIRC:
		components, err = parseSVGCircle(svg)
	default:
		Log.Error("Currently doesn't support shape type [%s]", shapeType)
		err = shared.InvalidShapeSvgStringError(svg)
	}

	return
}

func parseSVGCircle(svg string) (components []Component, err error) {
	tokens := strings.Split(svg, " ")

	if len(tokens) != 2 && len(tokens) != 6 {
		err = shared.InvalidShapeSvgStringError(svg)
		return
	}

	circle := Circle{0, 0, 0}

	for i := 0; i < len(tokens); i += 2 {
		if len(tokens) <= i+1 || err != nil {
			err = shared.InvalidShapeSvgStringError(svg)
			return
		}

		var val float64
		val, err = strconv.ParseFloat(tokens[i+1], 64)

		switch tokens[i] {
		case "cx":
			circle.X = val
		case "cy":
			circle.Y = val
		case "r":
			circle.R = val
		default:
			err = shared.InvalidShapeSvgStringError(svg)
		}
	}

	components = []Component{circle}
	return
}

// Parses SVG Path string into Components
func parseSVGPath(svg string) (components []Component, err error) {
	tokens := strings.Split(svg, " ")
	points := make(AdjacencyMap)

	// SVG must start with "M"
	if tokens[0] != "M" {
		err = shared.InvalidShapeSvgStringError(svg)
		return
	}

	prevPoint, lastM := Point{}, Point{}

	for i := 0; i < len(tokens); i++ {
		newPoint, offset, connected, valid := parseCommand(tokens, i, prevPoint, &lastM)
		if !valid {
			err = shared.InvalidShapeSvgStringError(svg)
			return
		}

		// Do not add duplicate consecutive points
		if !prevPoint.Equals(newPoint) {
			_, exists := points[newPoint]
			if !exists {
				points[newPoint] = []Point{}
			}
			if connected {
				points[newPoint] = append(points[newPoint], prevPoint)
				points[prevPoint] = append(points[prevPoint], newPoint)
			}
			prevPoint = newPoint
		}

		i += offset
	}

	// Extract single points and groups from the points map
	components = append(components, removeDisconnectedPoints(points)...)
	components = append(components, removeGroups(points)...)

	return
}

// Extract single points and remove them from points map
func removeDisconnectedPoints(points AdjacencyMap) []Component {
	toRemove := []Component{}
	for point, neighbours := range points {
		if len(neighbours) == 0 {
			toRemove = append(toRemove, point)
		}
	}

	for _, point := range toRemove {
		delete(points, point.(Point))
	}

	return toRemove
}

// Extract groups and remove them from points map
func removeGroups(points AdjacencyMap) []Component {
	removeDisconnectedPoints(points)

	if len(points) == 0 {
		return []Component{}
	}

	var point Point
	for point, _ = range points {
		// get a point from points
		break
	}
	// start with a point and trace the connected segments
	segments := removeGroup(point, points)
	group := Group{segments}

	return append(removeGroups(points), group)
}

// Traces edges connected to a point and extract a path
func removeGroup(curr Point, points AdjacencyMap) []Segment {
	neighbours, exists := points[curr]
	if !exists || len(neighbours) == 0 {
		return []Segment{}
	}

	// remove next point from curr point's neighbour list
	next := popNeighbour(&neighbours)
	points[curr] = neighbours

	// remove curr point from next point's neighbour list
	nextNeighbours := points[next]
	removeNeighbour(&nextNeighbours, curr)
	points[next] = nextNeighbours

	// create a segment witih curr and next as endpoints
	segment := NewSegment(next, curr)

	// recurse on next point
	return append(removeGroup(next, points), segment)
}

// Parses a command and returns new point and related info
func parseCommand(tokens []string, i int, prev Point, lastM *Point) (new Point, offset int, connected bool, valid bool) {
	numTokens, relative := len(tokens), false
	valid = false

	if strings.ToLower(tokens[i]) == tokens[i] {
		relative = true
	}

	switch tokens[i] {
	case "Z", "z":
		new = *lastM
		connected, offset, valid = true, 0, true
	case "M", "m":
		if i+2 < numTokens {
			new = createPoint(tokens[i+1], tokens[i+2], relative, prev)
			*lastM = new
			connected, offset, valid = false, 2, true
		}
	case "L", "l":
		if i+2 < numTokens {
			new = createPoint(tokens[i+1], tokens[i+2], relative, prev)
			connected, offset, valid = true, 2, true
		}
	case "H", "h":
		if i+1 < numTokens {
			new = createPoint(tokens[i+1], "", relative, prev)
			connected, offset, valid = true, 1, true
		}
	case "V", "v":
		if i+1 < numTokens {
			new = createPoint("", tokens[i+1], relative, prev)
			connected, offset, valid = true, 1, true
		}
	}

	return
}

// Creates a new point
func createPoint(xStr, yStr string, relative bool, prev Point) (newPoint Point) {
	if xStr == "" && yStr == "" {
		return
	}

	var x, y float64
	var err error

	if xStr == "" {
		x = prev.X
	} else {
		x, err = strconv.ParseFloat(xStr, 64)
		if err != nil {
			return
		}
		if relative {
			x += prev.X
		}
	}

	if yStr == "" {
		y = prev.Y
	} else {
		y, err = strconv.ParseFloat(yStr, 64)
		if err != nil {
			return
		}
		if relative {
			y += prev.Y
		}
	}

	newPoint = Point{x, y}
	return
}

// Remove and return the first neighbour
func popNeighbour(neighbours *[]Point) Point {
	size := len(*neighbours)

	if size == 0 {
		return Point{}
	}

	next := (*neighbours)[0]
	if size == 1 {
		*neighbours = []Point{}
	} else {
		*neighbours = (*neighbours)[1:]
	}

	return next
}

// Remove the specified neighbour
func removeNeighbour(neighbours *[]Point, toRemove Point) {
	var index int
	var neighbour Point
	for index, neighbour = range *neighbours {
		if neighbour.Equals(toRemove) {
			// get index
			break
		}
	}

	if len(*neighbours) > index+1 {
		*neighbours = append((*neighbours)[0:index], (*neighbours)[index+1:]...)
	} else {
		*neighbours = (*neighbours)[0:index]
	}
}
