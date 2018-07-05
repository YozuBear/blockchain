package miner

import (
	"../miner"
	"../shared"
	"reflect"
	"testing"
)

const (
	assertMsg = "Assert failed. Expected [%v], Actual [%v]"

	SingleLineSVG    = "M 20 10 L 30 40"
	TwoLinesSVG      = "M 1 1 v 20 m 10 0 l 10 -15"
	LinePointLineSVG = "M 0 0 H 20 m 0 10 L 0 0 m 0 10 m 5 5 L 15 30"
	TwoTrianglesSVG  = "M 5 20 L 10 30 L 0 35 Z M 3 8 L 7 5 L 2 5 Z"
	NonConvexPolySVG = "M 1 1 h 130 L 101 41 h -100 v -20 h 60 v -10 h -60 z"
	HourGlassSVG     = "M 5 5 L 0 10 L 10 10 L 5 15 z"
)

func TestParseSVGString(t *testing.T) {
	shapeType := shared.PATH
	var svg string
	var actual, expected []miner.Component
	var err error

	t.Run("SingleLine", func(t *testing.T) {
		svg = SingleLineSVG
		s1 := miner.NewSegment(miner.Point{20, 10}, miner.Point{30, 40})
		expected = []miner.Component{miner.Group{[]miner.Segment{s1}}}
		actual, err = ParseSVGString(svg, shapeType)
		if err != nil {
			t.Error(err)
		} else if !reflect.DeepEqual(actual, expected) {
			//t.Errorf(assertMsg, expected, actual)
		}
	})

	t.Run("TwoLines", func(t *testing.T) {
		svg = TwoLinesSVG
		actual, err = ParseSVGString(svg, shapeType)
		if err != nil {
			t.Fatal(err)
		}

		s1 := miner.NewSegment(miner.Point{1, 1}, miner.Point{1, 21})
		s2 := miner.NewSegment(miner.Point{11, 21}, miner.Point{21, 6})
		expected = []miner.Component{miner.Group{[]miner.Segment{s2}},
			miner.Group{[]miner.Segment{s1}}}
		if !reflect.DeepEqual(expected, actual) {
			//t.Errorf(assertMsg, expected, actual)
		}

	})

	t.Run("LinePointLine", func(t *testing.T) {
		svg = LinePointLineSVG
		actual, err = ParseSVGString(svg, shapeType)
		if err != nil {
			t.Fatal(err)
		}
		if len(actual) != 3 && len(actual) != 4 {
			// TODO connected non-loop segments may get split up
			t.Errorf(assertMsg, 3, len(actual))
		}
	})

	t.Run("TwoTriangles", func(t *testing.T) {
		svg = TwoTrianglesSVG
		actual, err = ParseSVGString(svg, shapeType)
		if err != nil {
			t.Fatal(err)
		}
		if len(actual) != 2 {
			t.Errorf(assertMsg, 2, len(actual))
		}
	})

	t.Run("NonConvexPolygon", func(t *testing.T) {
		svg = NonConvexPolySVG
		actual, err = ParseSVGString(svg, shapeType)
		if err != nil {
			t.Fatal(err)
		}
		if len(actual) != 1 || len(actual[0].(miner.Group).Segments) != 8 {
			t.Errorf(assertMsg, []int{1, 8}, []int{len(actual), len(actual[0].(miner.Group).Segments)})
		}
	})

	t.Run("Hourglass", func(t *testing.T) {
		svg = HourGlassSVG
		actual, err = ParseSVGString(svg, shapeType)
		if err != nil {
			t.Fatal(err)
		}
		if len(actual) != 1 || len(actual[0].(miner.Group).Segments) != 4 {
			t.Errorf(assertMsg, []int{1, 4}, []int{len(actual), len(actual[0].(miner.Group).Segments)})
		}
	})
}

func TestPoint(t *testing.T) {
	t.Run("IntersectsBorder", func(t *testing.T) {
		point := miner.Point{3, 4}

		seg1 := miner.NewSegment(miner.Point{1, 1}, miner.Point{10, 12})
		group := miner.Group{[]miner.Segment{seg1}}

		intersects := point.IntersectsBorder(group)
		if intersects {
			t.Errorf(assertMsg, false, intersects)
		}

		seg2 := miner.NewSegment(miner.Point{3, 4}, miner.Point{1, 2})
		group = miner.Group{[]miner.Segment{seg1, seg2}}
		intersects = point.IntersectsBorder(group)
		if !intersects {
			t.Errorf(assertMsg, true, intersects)
		}

		seg3 := miner.NewSegment(miner.Point{2, 7}, miner.Point{100, 287})
		group = miner.Group{[]miner.Segment{seg1, seg3}}
		if !intersects {
			t.Errorf(assertMsg, true, intersects)
		}
	})

	t.Run("IsBetween", func(t *testing.T) {
		p1 := miner.Point{1, 1}
		p2 := miner.Point{4, 5}
		p3 := miner.Point{8, 9}

		result := p2.IsBetween(p1, p3)
		if !result {
			t.Errorf(assertMsg, true, result)
		}
		result = p2.IsBetween(p3, p1)
		if !result {
			t.Errorf(assertMsg, true, result)
		}
	})
}

func TestGroup(t *testing.T) {
	shapeType := shared.PATH

	svg := TwoTrianglesSVG
	twoTriangles, err := ParseSVGString(svg, shapeType)
	if err != nil {
		t.Fatal(err)
	}
	if len(twoTriangles) != 2 {
		t.Fatal(err)
	}
	triangle1 := twoTriangles[0] //TODO if flipped, would affect test results
	triangle2 := twoTriangles[1]

	svg = NonConvexPolySVG
	nonConvexPolygon, err := ParseSVGString(svg, shapeType)
	if err != nil {
		t.Fatal(err)
	}
	polygon := nonConvexPolygon[0]

	svg = HourGlassSVG
	hourglassShape, err := ParseSVGString(svg, shapeType)
	if err != nil {
		t.Fatal(err)
	}
	hourglass := hourglassShape[0]

	var point miner.Point

	t.Run("ContainsPoint", func(t *testing.T) {
		point = miner.Point{3, 5} // in interior of triangle1
		contains := triangle1.ContainsPoint(point, true)
		if !contains {
			t.Errorf(assertMsg, true, contains)
		}

		contains = triangle1.ContainsPoint(point, false)
		if contains {
			t.Errorf(assertMsg, false, contains)
		}

		contains = triangle2.ContainsPoint(point, true)
		if contains {
			t.Errorf(assertMsg, false, contains)
		}

		point = miner.Point{0, 35} // vertex of triangle2
		contains = triangle2.ContainsPoint(point, true)
		if contains {
			t.Errorf(assertMsg, false, contains)
		}

		point = miner.Point{2, 34} // on border of triangle2
		contains = triangle2.ContainsPoint(point, true)
		if contains {
			t.Errorf(assertMsg, false, contains)
		}

		point = miner.Point{4, 30} // interior of triangle2
		contains = triangle2.ContainsPoint(point, true)
		if !contains {
			t.Errorf(assertMsg, true, contains)
		}

		point = miner.Point{80, 15} // interior of polygon
		contains = polygon.ContainsPoint(point, true)
		if !contains {
			t.Errorf(assertMsg, true, contains)
		}

		point = miner.Point{30, 11} // between folds of polygon
		contains = polygon.ContainsPoint(point, true)
		if contains {
			t.Errorf(assertMsg, false, contains)
		}
	})

	t.Run("IntersectsBorder", func(t *testing.T) {
		intersects := polygon.IntersectsBorder(miner.Point{1, 2}) // on segment
		if !intersects {
			t.Errorf(assertMsg, true, intersects)
		}

		intersects = polygon.IntersectsBorder(miner.Point{101, 41}) // vertex
		if !intersects {
			t.Errorf(assertMsg, true, intersects)
		}

		p1, p2, p3 := miner.Point{58, 25}, miner.Point{50, 17}, miner.Point{60, 17}
		s1, s2, s3 := miner.NewSegment(p1, p2), miner.NewSegment(p2, p3), miner.NewSegment(p3, p1)
		intersects = polygon.IntersectsBorder(miner.Group{[]miner.Segment{s1, s2, s3}})
		if !intersects {
			t.Errorf(assertMsg, true, intersects)
		}

		p1, p2, p3 = miner.Point{58, 20}, miner.Point{50, 12}, miner.Point{60, 12}
		s1, s2, s3 = miner.NewSegment(p1, p2), miner.NewSegment(p2, p3), miner.NewSegment(p3, p1)
		intersects = polygon.IntersectsBorder(miner.Group{[]miner.Segment{s1, s2, s3}})
		if intersects {
			t.Errorf(assertMsg, false, intersects)
		}
	})

	t.Run("Area", func(t *testing.T) {
		area := polygon.Area(true, false)
		if area != 4000 {
			t.Errorf(assertMsg, 4000, area)
		}

		area = polygon.Area(false, true)
		if area != 440 {
			t.Errorf(assertMsg, 440, area)
		}

		area = polygon.Area(true, true)
		if area != 4440 {
			t.Errorf(assertMsg, 4440, area)
		}
	})

	t.Run("IsSimpleClosed", func(t *testing.T) {
		result := polygon.IsSimpleClosed()
		if !result {
			t.Errorf(assertMsg, true, result)
		}

		result = triangle1.IsSimpleClosed()
		if !result {
			t.Errorf(assertMsg, true, result)
		}

		result = triangle2.IsSimpleClosed()
		if !result {
			t.Errorf(assertMsg, true, result)
		}

		result = hourglass.IsSimpleClosed() // self intersects
		if result {
			t.Errorf(assertMsg, false, result)
		}

		s1 := miner.NewSegment(miner.Point{0, 0}, miner.Point{10, 20})
		s2 := miner.NewSegment(miner.Point{10, 20}, miner.Point{2, 3})
		comp := miner.Group{[]miner.Segment{s1, s2}} // not closed
		result = comp.IsSimpleClosed()
		if result {
			t.Errorf(assertMsg, false, result)
		}
	})
}

func TestShape(t *testing.T) {
	canvas := miner.Canvas{make(map[string]miner.Shape), 1500, 1000}

	shape := miner.Shape{
		"ownerhash",
		shared.PATH,
		"", "", "black",
		[]miner.Component{},
	}

	shapeType := shared.PATH

	svg := SingleLineSVG
	singleLine, err := ParseSVGString(svg, shapeType)
	if err != nil {
		t.Fatal(err)
	}
	if len(singleLine) != 1 {
		t.Fatalf(assertMsg, 1, len(singleLine))
	}

	svg = TwoTrianglesSVG
	twoTriangles, err := ParseSVGString(svg, shapeType)
	if err != nil {
		t.Fatal(err)
	}
	if len(twoTriangles) != 2 {
		t.Fatal(err)
	}

	svg = NonConvexPolySVG
	nonConvexPolygon, err := ParseSVGString(svg, shapeType)
	if err != nil {
		t.Fatal(err)
	}

	svg = HourGlassSVG
	hourglass, err := ParseSVGString(svg, shapeType)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("WithinCanvas", func(t *testing.T) {
		point := miner.Point{0, 0}
		shape.Components = []miner.Component{point}
		result := shape.WithinCanvas(canvas)
		if !result {
			t.Errorf(assertMsg, true, result)
		}

		s1 := miner.NewSegment(miner.Point{1501, 100}, miner.Point{40, 20})
		s2 := miner.NewSegment(miner.Point{3, 4}, miner.Point{5, 6})
		shape.Components = []miner.Component{miner.Group{[]miner.Segment{s1, s2}}, point}
		result = shape.WithinCanvas(canvas)
		if result {
			t.Errorf(assertMsg, false, result)
		}
	})

	t.Run("IsFillValid", func(t *testing.T) {
		shape.Components = hourglass
		shape.Fill = "blue"
		result := shape.IsFillValid()
		if result {
			t.Errorf(assertMsg, false, result)
		}

		shape.Components = hourglass
		shape.Fill = "transparent"
		result = shape.IsFillValid()
		if !result {
			t.Errorf(assertMsg, true, result)
		}

		shape.Components = nonConvexPolygon
		shape.Fill = "blue"
		result = shape.IsFillValid()
		if !result {
			t.Errorf(assertMsg, true, result)
		}

		shape.Components = twoTriangles
		shape.Fill = "black"
		result = shape.IsFillValid()
		if result {
			t.Errorf(assertMsg, false, result)
		}

		shape.Components = twoTriangles
		shape.Fill = "transparent"
		result = shape.IsFillValid()
		if !result {
			t.Errorf(assertMsg, true, result)
		}

		shape.Components = singleLine
		shape.Fill = "red"
		result = shape.IsFillValid()
		if result {
			t.Errorf(assertMsg, false, result)
		}
	})

	t.Run("IllegalOverlap", func(t *testing.T) {
		polygonFilledShape := miner.Shape{
			NonConvexPolySVG, shapeType,
			NonConvexPolySVG, "red", "red",
			nonConvexPolygon}
		polygonEmptyShape := miner.Shape{
			NonConvexPolySVG, shapeType,
			NonConvexPolySVG, "transparent", "red",
			nonConvexPolygon}

		canvas.Shapes[NonConvexPolySVG] = polygonEmptyShape

		singleLineShape := miner.Shape{
			SingleLineSVG, shapeType,
			SingleLineSVG, "transparent", "blue",
			singleLine}

		result := singleLineShape.IllegalOverlap(canvas)
		if !result {
			t.Errorf(assertMsg, true, result)
		}

		group := miner.Group{[]miner.Segment{miner.NewSegment(miner.Point{80, 30}, miner.Point{120, 8})}}
		segmentShape := miner.Shape{
			"segmentShape", shapeType,
			"svg", "transparent", "red",
			[]miner.Component{group}}
		result = segmentShape.IllegalOverlap(canvas)
		if result {
			t.Errorf(assertMsg, false, result)
		}

		canvas.Shapes[NonConvexPolySVG] = polygonFilledShape
		result = segmentShape.IllegalOverlap(canvas)
		if !result {
			t.Errorf(assertMsg, true, result)
		}

	})

	t.Run("Area", func(t *testing.T) {
		twoTrianglesShape := miner.Shape{
			TwoTrianglesSVG, shapeType,
			TwoTrianglesSVG, "transparent", "red",
			twoTriangles}

		area, _ := twoTrianglesShape.Area()
		if area != 52 {
			t.Errorf(assertMsg, 52, area)
		}
	})

	canvas = miner.Canvas{make(map[string]miner.Shape), 1500, 1000}
	t.Run("Validate", func(t *testing.T) {
		/* Add shape 1 */
		svg, fill, stroke := "M 10 10 v 5 h 8 z", "red", "blue"
		comps, err := ParseSVGString(svg, shared.PATH)
		if err != nil {
			t.Fatal(err)
		}
		shape := makeShape("owner1", svg, fill, stroke, comps)
		if err = shape.Validate(canvas); err != nil {
			t.Fatal(err)
		} else {
			canvas.Shapes[svg] = shape
		}

		/* Try to add shape 1 again with different owner -> should fail */
		svg, fill, stroke = "M 10 10 v 5 h 8 z", "green", "yellow"
		comps, err = ParseSVGString(svg, shared.PATH)
		if err != nil {
			t.Fatal(err)
		}
		shape = makeShape("owner2", svg, fill, stroke, comps)
		if err = shape.Validate(canvas); err == nil {
			t.Fatalf("expected illegalOverlapError but received nil")
		}

		/* Try to add transparent stroke and fill */
		svg, fill, stroke = "M 10 10 v 5 h 8 z", "transparent", "transparent"
		comps, err = ParseSVGString(svg, shared.PATH)
		if err != nil {
			t.Fatal(err)
		}
		shape = makeShape("owner2", svg, fill, stroke, comps)
		if err = shape.Validate(canvas); err == nil {
			t.Fatalf("expected invalidsvg but received nil")
		}
	})
}

func TestCircle(t *testing.T) {
	t.Run("Parse", func(t *testing.T) {
		svg := "cy 4 3 r 0 cx 5"
		comps, err := ParseSVGString(svg, shared.CIRC)
		if err == nil {
			t.Fatalf("expected error")
		}

		svg = "cy 5 r 10 cx 4"
		comps, err = ParseSVGString(svg, shared.CIRC)

		if err != nil || len(comps) != 1 {
			t.Fatalf("cy 5 r 10 cx 4 bad result %v, %v", err, comps)
		}

		out := comps[0].(miner.Circle)
		circle := miner.Circle{10, 4, 5}
		if circle.X != out.X || circle.Y != out.Y || circle.R != out.R {
			t.Fatalf(assertMsg, circle, out)
		}

		svg = "r 4"
		comps, err = ParseSVGString(svg, shared.CIRC)

		if err != nil || len(comps) != 1 {
			t.Fatalf("r4 bad result %v, %v", err, comps)
		}

		out = comps[0].(miner.Circle)
		circle = miner.Circle{4, 0, 0}
		if circle.X != out.X || circle.Y != out.Y || circle.R != out.R {
			t.Fatalf(assertMsg, circle, out)
		}
	})

	shape1 := miner.Shape{
		Owner:      "owner1",
		ShapeType:  shared.CIRC,
		Svg:        "",
		Fill:       "",
		Stroke:     "",
		Components: []miner.Component{},
	}

	t.Run("Area", func(t *testing.T) {
		svg := "r 5 cx 7 cy 9"
		comps, err := ParseSVGString(svg, shared.CIRC)
		if err != nil {
			t.Fatalf("not expecting error but got %s", err.Error())
		}
		shape1.Components = comps
		shape1.Svg = svg

		shape1.Fill = "red"
		shape1.Stroke = "transparent"

		area1, err := shape1.Area()
		if err != nil {
			t.Fatal(err)
		}
		if area1 != 79 {
			t.Errorf(assertMsg, 79, area1)
		}

		shape1.Fill = "red"
		shape1.Stroke = "blue"

		area1, err = shape1.Area()
		if err != nil {
			t.Fatal(err)
		}
		if area1 != 110 {
			t.Errorf(assertMsg, 110, area1)
		}
	})

	canvas := miner.Canvas{make(map[string]miner.Shape), 1000, 1500}

	t.Run("ValidateOverlap", func(t *testing.T) {

		shape1.Fill = "transparent"
		canvas.Shapes["shape1"] = shape1

		// segment crosses circle
		shape, err := makePATHShape("M 0 0 L 50 50", "transparent", "red", "owner2")
		if err != nil {
			t.Fatal(err)
		}
		if !isIllegalOverlap(shape.Validate(canvas)) {
			t.Fatal("expecting illegal overlap")
		}

		// one intersection
		shape, err = makePATHShape("M 2 0 v 100", "transparent", "red", "owner2")
		if err != nil {
			t.Fatal(err)
		}
		if !isIllegalOverlap(shape.Validate(canvas)) {
			t.Fatal("expecting illegal overlap")
		}

		// outside of circle
		shape, err = makePATHShape("M 15 15 L 50 50", "transparent", "red", "owner2")
		if err != nil {
			t.Fatal(err)
		}
		err = shape.Validate(canvas)
		if isIllegalOverlap(err) {
			t.Fatalf("not expecting illegal overlap, got %v", err)
		}

		// seg inside empty circle
		shape, err = makePATHShape("M 6 6 L 10 10", "transparent", "red", "owner2")
		if err != nil {
			t.Fatal(err)
		}
		err = shape.Validate(canvas)
		if isIllegalOverlap(err) {
			t.Fatalf("not expecting illegal overlap, got %v", err)
		}

		// circ inside empty circle
		shape, err = makeCIRCShape("cx 8 cy 9 r 1", "transparent", "red", "owner2")
		if err != nil {
			t.Fatal(err)
		}
		err = shape.Validate(canvas)
		if isIllegalOverlap(err) {
			t.Fatalf("not expecting illegal overlap, got %v", err)
		}

		// FILLED CIRCLE

		shape1.Fill = "red"
		canvas.Shapes["shape1"] = shape1

		// inside of filled circle
		shape, err = makePATHShape("M 6 6 L 10 10", "transparent", "red", "owner2")
		if err != nil {
			t.Fatal(err)
		}
		if !isIllegalOverlap(shape.Validate(canvas)) {
			t.Fatal("expecting illegal overlap")
		}

		// cirlce intersecting circle
		shape, err = makeCIRCShape("cx 15 cy 10 r 8", "transparent", "red", "owner2")
		if err != nil {
			t.Fatal(err)
		}
		if !isIllegalOverlap(shape.Validate(canvas)) {
			t.Fatal("expecting illegal overlap")
		}
	})

	t.Run("ValidateBoundary", func(t *testing.T) {
		shape, err := makeCIRCShape("cx 2 cy 3 r 5", "transparent", "red", "owner1")
		if err != nil {
			t.Fatal(err)
		}
		// canvas contains owner1 shapes only
		err = shape.Validate(canvas)
		if err == nil {
			t.Fatalf("expecting out of bounds error")
		}
		switch err.(type) {
		case shared.OutOfBoundsError:
		default:
			t.Fatal(err)
		}
	})
}

func TestShapeHash(t *testing.T) {
	svg, fill, stroke := "M 1 2 L 3 4", "transparentz", "purple"
	comps, _ := ParseSVGString(svg, shared.PATH)
	shape := makeShape("owner2", svg, fill, stroke, comps)
	h1 := shape.HashToString()

	svg, fill, stroke = "M 400 500 v 1", "transparent", "green"
	comps, _ = ParseSVGString(svg, shared.PATH)
	shape = makeShape("owner2", svg, fill, stroke, comps)
	h2 := shape.HashToString()

	if h1 == h2 {
		t.Errorf("shape hashes should not equal")
	}
}

func makeShape(owner, svg, fill, stroke string, comps []miner.Component) miner.Shape {
	return miner.Shape{owner, shared.PATH, svg, fill, stroke, comps}
}

func ParseSVGString(svg string, sType shared.ShapeType) ([]miner.Component, error) {
	return miner.ParseSVGString(svg, sType)
}
func makePATHShape(svg, fill, stroke, owner string) (shape miner.Shape, err error) {
	return makeSVGShape(svg, fill, stroke, owner, shared.PATH)
}

func makeCIRCShape(svg, fill, stroke, owner string) (shape miner.Shape, err error) {
	return makeSVGShape(svg, fill, stroke, owner, shared.CIRC)
}

func makeSVGShape(svg, fill, stroke, owner string, stype shared.ShapeType) (shape miner.Shape, err error) {
	comps, err := ParseSVGString(svg, stype)
	if err != nil {
		return
	}

	shape.Fill = fill
	shape.Stroke = stroke
	shape.Owner = owner
	shape.Components = comps
	return
}

func isIllegalOverlap(e error) bool {
	if e == nil {
		return false
	}

	switch e.(type) {
	case shared.ShapeOverlapError:
		return true
	}
	return false
}
