package miner

import (
	"../shared"
	"encoding/hex"
	"math"
)

type Shape struct {
	Owner             string // public key hash of the shape owner
	ShapeType         shared.ShapeType
	Svg, Fill, Stroke string // addshape args
}

func (s Shape) Area() (uint64, error) {
	components, err := ParseSVGString(s.Svg, s.ShapeType)

	if err != nil {
		return 0, err
	}

	if !s.IsFillValid(components) {
		return 0, shared.InvalidShapeSvgStringError(s.Svg)
	}

	var area float64
	for _, comp := range components {
		area += comp.Area(s.IsFilled(), s.HasStroke())
	}

	return uint64(math.Ceil(area)), nil
}

// validates a shape against the canvas
func (s Shape) Validate(canvas Canvas) error {
	components, err := ParseSVGString(s.Svg, s.ShapeType)
	if err != nil {
		return err
	}

	if !s.WithinCanvas(canvas, components) {
		return shared.OutOfBoundsError{}
	}

	if (!s.HasStroke() && !s.IsFilled()) || !s.IsFillValid(components) {
		return shared.InvalidShapeSvgStringError(s.Svg)
	}

	if s.IllegalOverlap(canvas, components) {
		return shared.ShapeOverlapError(s.Svg)
	}

	return nil
}

// checks if the shape is in the canvas boundary
func (s Shape) WithinCanvas(canvas Canvas, components []Component) bool {
	xMax := float64(canvas.XMax)
	yMax := float64(canvas.YMax)

	for _, comp := range components {
		if !comp.IsBoundedBy(0, xMax, 0, yMax) {
			return false
		}
	}

	return true
}

// checks if shape is ONE simple closed curve
func (s Shape) IsFillValid(components []Component) bool {
	if !s.IsFilled() {
		return true
	}

	// only allow ONE simple closed curve
	if len(components) != 1 {
		return false
	}

	if comp := components[0]; !comp.IsSimpleClosed() {
		return false
	}
	return true
}

// checks if the shape overlaps with another shape with a different owner
func (s Shape) IllegalOverlap(canvas Canvas, newComps []Component) bool {
	// XXX optimize: only check shapes in nearby regions of the canvas
	for _, oldShape := range canvas.Shapes {
		if oldShape.Owner == s.Owner {
			continue
		}

		oldFill, newFill := oldShape.IsFilled(), s.IsFilled()
		oldComps, _ := ParseSVGString(oldShape.Svg, oldShape.ShapeType)

		for _, oldComp := range oldComps {
			for _, newComp := range newComps {
				if newComp.IntersectsBorder(oldComp) ||
					newComp.ContainsPoint(oldComp.GetPoint(), newFill) ||
					oldComp.ContainsPoint(newComp.GetPoint(), oldFill) {
					return true
				}
			}
		}
	}

	return false
}

// checks if a shape is filled
func (s Shape) IsFilled() bool {
	return s.Fill != "transparent"
}

func (s Shape) HasStroke() bool {
	return s.Stroke != "transparent"
}

// Hash Shape's fields to string
// Fields: owner, svg, fill, stroke, components
func (s Shape) HashToString() (hashedShape string) {
	shapeBytes := s.HashToBytes()
	return hex.EncodeToString(shapeBytes[:])
}

// Hash Shape's fields
// Fields: owner, svg, fill, stroke, components
func (s Shape) HashToBytes() (hashedShape []byte) {
	return shared.HashByteArr(shared.Serialize(s.Svg + s.Fill + s.Stroke))
}

// Returns the svg, fill, stroke of the shape
func (s Shape) GetSVGFields() string {
	str := s.Svg + "," + s.Fill + "," + s.Stroke
	switch s.ShapeType {
	case shared.PATH:
		str = "PATH," + str
	case shared.CIRC:
		str = "CIRC," + str
	default:
		Log.Error("Bad ShapeType")
	}
	return str
}

func (s *Shape) EraseShape() {
	s.Fill = "white"
	s.Stroke = "white"
}
