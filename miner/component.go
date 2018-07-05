package miner

import (
	"math"
	"reflect"
)

type Component interface {
	// check if c intersects with this on the border
	IntersectsBorder(c Component) bool

	// calculate the area of this in pixels
	Area(fill, stroke bool) float64

	// checks if point is in the filled interior of this
	ContainsPoint(point Point, fill bool) bool

	// checks if this is in the bounding box
	IsBoundedBy(xMin, xMax, yMin, yMax float64) bool

	// checks if this is a simple closed curve
	IsSimpleClosed() bool

	// Returns a point in this
	GetPoint() Point
}

type Point struct {
	X, Y float64
}

type Group struct {
	Segments []Segment
}

type Circle struct {
	R, X, Y float64
}

type Segment struct {
	PStart, PEnd Point
	YInt, Slope  float64
}

var (
	PointType = reflect.TypeOf(Point{})
	GroupType = reflect.TypeOf(Group{})

	epsilon = 0.000001 // preset floating point precision
)

/* Point Implementation of Component */

func (p Point) IntersectsBorder(c Component) (intersects bool) {
	switch c.(type) {
	case Point:
		intersects = p.Equals(c.(Point))
	case Group:
		for _, segment := range c.(Group).Segments {
			if segment.IntersectsPoint(p) {
				intersects = true
				break
			}
		}
	case Circle:
		intersects = c.(Circle).IntersectsPoint(p)
	}
	return
}

func (p Point) Area(fill, stroke bool) float64 {
	return 0
}

func (p Point) ContainsPoint(point Point, fill bool) bool {
	return p.Equals(point)
}

func (p Point) IsBoundedBy(xMin, xMax, yMin, yMax float64) bool {
	return p.X >= xMin && p.X <= xMax && p.Y >= yMin && p.Y <= yMax
}

func (p Point) IsSimpleClosed() bool {
	return false
}

func (p Point) GetPoint() Point {
	return p
}

func (p Point) Equals(point Point) bool {
	return FloatEQ(p.X, point.X) && FloatEQ(p.Y, point.Y)
}

func (p Point) IsBetween(p1 Point, p2 Point) bool {
	return ((FloatLEQ(p.X, p1.X) && FloatGEQ(p.X, p2.X)) || (FloatLEQ(p.X, p2.X) && FloatGEQ(p.X, p1.X))) &&
		((FloatLEQ(p.Y, p1.Y) && FloatGEQ(p.Y, p2.Y)) || (FloatLEQ(p.Y, p2.Y) && FloatGEQ(p.Y, p1.Y)))
}

/* Group Implementation of Component */

func (g Group) IntersectsBorder(c Component) (intersects bool) {
	switch c.(type) {
	case Point:
		intersects = c.(Point).IntersectsBorder(g)
	case Group:
		// check if a pair of segments in g intersect each other
		for _, s1 := range c.(Group).Segments {
			for _, s2 := range g.Segments {
				if s1.IntersectsWith(s2) {
					intersects = true
					break
				}
			}
		}
	case Circle:
		intersects = c.(Circle).IntersectsBorder(g)
	}

	return intersects
}

func (g Group) Area(fill, stroke bool) (area float64) {
	if fill {
		// Green's Theorem: A(shape) = abs((1/2)Sum(x[i]*y[i+1] - y[i]*x[i+1]))
		for _, s := range g.Segments {
			area += s.PStart.X*s.PEnd.Y - s.PStart.Y*s.PEnd.X
		}

		area = math.Abs(area) / 2
	}

	if stroke {
		// A = sum of segment lengths
		for _, s := range g.Segments {
			area += s.Length()
		}
	}

	return
}

// assumes g is simple closed curve
func (g Group) ContainsPoint(point Point, fill bool) bool {
	if !fill {
		return false
	}

	// https://stackoverflow.com/questions/13217224/determining-if-a-set-of-points-are-inside-or-outside-a-square/13217508#13217508
	pOut := Point{-1, -1}
	testSeg := NewSegment(point, pOut)
	numIntersections := 0
	for _, seg := range g.Segments {
		if testSeg.IntersectsWith(seg) {
			numIntersections++
		}
	}

	// point is inside of closed curve if the test line crosses borders an odd #times
	return numIntersections%2 != 0
}

func (g Group) IsBoundedBy(xMin, xMax, yMin, yMax float64) bool {
	for _, segment := range g.Segments {
		if !segment.PStart.IsBoundedBy(xMin, xMax, yMin, yMax) ||
			!segment.PEnd.IsBoundedBy(xMin, xMax, yMin, yMax) {
			return false
		}
	}

	return true
}

func (g Group) IsSimpleClosed() bool {
	size := len(g.Segments)

	// Check if closed
	firstPoint := g.Segments[0].PStart
	lastPoint := g.Segments[size-1].PEnd
	if !firstPoint.Equals(lastPoint) {
		return false
	}

	// Check if self-intersects
	for i := 0; i < size; i++ {
		for j := i + 1; j < size; j++ {
			if g.Segments[i].Crosses(g.Segments[j]) {
				return false
			}
		}
	}

	return true
}

// returns any point in g
func (g Group) GetPoint() Point {
	return g.Segments[0].PStart
}

/* Circle implementation of Component */

func (c Circle) IntersectsBorder(comp Component) (intersects bool) {
	intersects = false

	switch comp.(type) {
	case Point:
		intersects = c.IntersectsPoint(comp.(Point))
	case Group:
		for _, seg := range comp.(Group).Segments {
			if c.IntersectsSegment(seg) {
				intersects = true
				break
			}
		}
	case Circle:
		intersects = c.IntersectsCircle(comp.(Circle))
	}
	return intersects
}

func (c Circle) Area(fill, stroke bool) (area float64) {
	if fill {
		area += c.R * c.R * math.Pi
	}

	if stroke {
		area += 2 * c.R * math.Pi
	}

	return
}

func (c Circle) ContainsPoint(point Point, fill bool) bool {
	if !fill {
		return false
	}

	// https://stackoverflow.com/questions/13217224/determining-if-a-set-of-points-are-inside-or-outside-a-square/13217508#13217508
	pOut := Point{-1, -1}
	testSeg := NewSegment(point, pOut)
	numIntersections := 0
	if c.IntersectsSegment(testSeg) {
		numIntersections++
	}

	// point is inside of closed curve if the test line crosses borders an odd #times
	return numIntersections%2 != 0
}

func (c Circle) IsBoundedBy(xMin, xMax, yMin, yMax float64) bool {
	circXMin, circXMax := c.X-c.R, c.X+c.R
	circYMin, circYMax := c.Y-c.R, c.Y+c.R
	return FloatLEQ(xMin, circXMin) && FloatGEQ(xMax, circXMax) &&
		FloatLEQ(yMin, circYMin) && FloatGEQ(yMax, circYMax)
}

func (c Circle) IsSimpleClosed() bool {
	return true
}

func (c Circle) GetPoint() Point {
	return Point{c.X, c.Y}
}

// checks if point is on the circumference of circle
func (c Circle) IntersectsPoint(p Point) bool {
	return FloatEQ(c.R, Distance(c.X, p.X, c.Y, p.Y))
}

// Checks if a segment intersects with a circle
func (c Circle) IntersectsSegment(seg Segment) bool {
	coeffA := 1 + Sqr(seg.Slope)
	coeffB := 2*seg.Slope*seg.YInt - 2*seg.Slope*c.Y - 2*c.X
	coeffC := c.X + Sqr(seg.YInt) - 2*seg.YInt*c.Y + Sqr(c.Y) - Sqr(c.R)

	term1 := -1 * coeffB
	term2 := Sqr(coeffB) - 4*coeffA*coeffC
	denom := 2 * coeffA

	if term2 < 0 {
		// no solution
		return false
	}

	x1 := (term1 + Sqrt(term2)) / denom
	y1 := seg.GetY(x1)
	if seg.IntersectsPoint(Point{x1, y1}) {
		return true
	}

	x2 := (term1 - Sqrt(term2)) / denom
	y2 := seg.GetY(x2)
	if seg.IntersectsPoint(Point{x2, y2}) {
		return true
	}

	return false
}

// https://stackoverflow.com/questions/3349125/circle-circle-intersection-points
// Checks if two circles intersect
func (c0 Circle) IntersectsCircle(c1 Circle) bool {
	d := Distance(c0.X, c1.X, c0.Y, c1.Y)
	r0, r1 := c0.R, c1.R

	// circles are separate, non touching; circle is within the other circle
	if !FloatLEQ(d, r0+r1) || !FloatGEQ(d, math.Abs(r0-r1)) {
		return false
	}

	return true
}

/* Segment helpers */

// creates a new segment (calculates yInt and slope)
func NewSegment(p1, p2 Point) Segment {
	deltaX := p2.X - p1.X
	if deltaX == 0 {
		// slope cannot be undefined; replace with 1/epsilon instead
		deltaX = epsilon
	}
	slope := (p2.Y - p1.Y) / deltaX
	yInt := p1.Y - slope*p1.X
	return Segment{p1, p2, yInt, slope}
}

// checks if a point is on the segment
func (s Segment) IntersectsPoint(p Point) bool {
	return p.IsBetween(s.PStart, s.PEnd) && FloatEQ((p.Y-s.YInt)/s.Slope, p.X)
}

// checks if two Segments intersect at any point
func (s Segment) IntersectsWith(s2 Segment) bool {
	if s.Slope == s2.Slope {
		return false
	}

	x := (s2.YInt - s.YInt) / (s.Slope - s2.Slope)
	y := s.GetY(x)
	p := &Point{x, y}

	return p.IsBetween(s.PStart, s.PEnd) && p.IsBetween(s2.PStart, s2.PEnd)
}

func (s Segment) GetY(x float64) float64 {
	return s.Slope*x + s.YInt
}

// checks if two segments cross each other at non endpoints
func (s Segment) Crosses(s2 Segment) bool {
	crosses := false
	if s.IntersectsWith(s2) {
		// check intersection isn't one of the endpoints
		if !s.PStart.Equals(s2.PStart) && !s.PStart.Equals(s2.PEnd) &&
			!s.PEnd.Equals(s2.PStart) && !s.PEnd.Equals(s2.PEnd) {
			crosses = true
		}
	}
	return crosses
}

// computes euclidean norm of a segment
func (s Segment) Length() float64 {
	return Distance(s.PStart.X, s.PEnd.X, s.PStart.Y, s.PEnd.Y)
}

/* math helpers */

func Sqrt(x float64) float64 {
	return math.Sqrt(x)
}

func Sqr(x float64) float64 {
	return math.Pow(x, 2)
}

func Distance(x1, x2, y1, y2 float64) float64 {
	return math.Sqrt(math.Pow(x1-x2, 2) + math.Pow(y1-y2, 2))
}

/* Floating point helpers */

func FloatEQ(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func FloatLEQ(a, b float64) bool {
	return a-epsilon <= b+epsilon
}

func FloatGEQ(a, b float64) bool {
	return FloatLEQ(b, a)
}
