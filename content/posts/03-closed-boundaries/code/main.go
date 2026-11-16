//lint:file-ignore U1000 benchmark results go to sinks that are never read, so the compiler keeps the work

package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"

	geom "github.com/gravitton/geometry"
)

const rectangles = 1_000_000

var sinkBool bool

func main() {
	oneCorner()
	cornersBuiltFrom()
	notANumber()
	touching()
	toleranceBand()
	integers()
	cost()
}

func oneCorner() {
	random := rand.New(rand.NewPCG(3, 4))
	for {
		low := geom.Pt(float64(random.IntN(100000))/100, float64(random.IntN(100000))/100)
		high := geom.Pt(low.X+float64(random.IntN(100000))/100, low.Y+float64(random.IntN(100000))/100)
		rectangle := geom.RectangleFromMinMax(low, high)
		if containsExactly(rectangle, low) {
			continue
		}

		topLeft := rectangle.TopLeft()
		fmt.Printf("a rectangle from (%v, %v) to (%v, %v):\n", low.X, low.Y, high.X, high.Y)
		fmt.Printf("  TopLeft:           (%v, %v)\n", topLeft.X, topLeft.Y)
		fmt.Printf("  TopLeft - low:     %g, %g\n", topLeft.X-low.X, topLeft.Y-low.Y)
		fmt.Println("  low inside by <=:  ", containsExactly(rectangle, low))
		fmt.Println("  low inside by Contains:", rectangle.Contains(low))

		return
	}
}

func cornersBuiltFrom() {
	random := rand.New(rand.NewPCG(1, 2))
	drifted, lostByExact, keptByContains := 0, 0, 0

	for range rectangles {
		low := geom.Pt(random.Float64()*1000, random.Float64()*1000)
		high := geom.Pt(low.X+random.Float64()*1000, low.Y+random.Float64()*1000)
		rectangle := geom.RectangleFromMinMax(low, high)

		if rectangle.TopLeft() != low || rectangle.BottomRight() != high {
			drifted++
		}
		if !containsExactly(rectangle, low) || !containsExactly(rectangle, high) {
			lostByExact++
		}
		if rectangle.Contains(low) && rectangle.Contains(high) {
			keptByContains++
		}
	}

	fmt.Println("rectangles built from two corners:", rectangles)
	fmt.Println("  TopLeft or BottomRight differs from the corner given:", drifted)
	fmt.Println("  a given corner is outside by exact <=:", lostByExact)
	fmt.Println("  both given corners inside by Contains:", keptByContains)
}

func containsExactly(rectangle geom.Rectangle[float64], point geom.Point[float64]) bool {
	low, high := rectangle.TopLeft(), rectangle.BottomRight()

	return point.X >= low.X && point.Y >= low.Y && point.X <= high.X && point.Y <= high.Y
}

func notANumber() {
	point := geom.Pt(math.NaN(), 5.0)
	rectangle := geom.RectangleFromMinMax(geom.Pt(0.0, 0.0), geom.Pt(10.0, 10.0))
	circle := geom.Circ(geom.Pt(5.0, 5.0), 5.0)
	rejectedOutside := !(point.X < 0 || point.X > 10 || point.Y < 0 || point.Y > 10)

	fmt.Println("a point with a NaN coordinate:")
	fmt.Println("  inside by \"not outside\":", rejectedOutside)
	fmt.Println("  inside the rectangle:", rectangle.Contains(point))
	fmt.Println("  inside the circle:", circle.Contains(point))
	fmt.Println("  inside the polygon:", rectangle.Polygon().Contains(point))
}

func touching() {
	left := geom.RectangleFromMinMax(geom.Pt(0.0, 0.0), geom.Pt(10.0, 10.0))
	right := geom.RectangleFromMinMax(geom.Pt(10.0, 0.0), geom.Pt(20.0, 10.0))

	fmt.Println("two rectangles sharing an edge intersect:", left.IntersectsRectangle(right))
	shared, ok := left.ClipRectangle(right)
	fmt.Println("  their common part:", ok, shared.Size, "area", shared.Area())
}

func toleranceBand() {
	rectangle := geom.RectangleFromMinMax(geom.Pt(0.0, 0.0), geom.Pt(10.0, 10.0))

	fmt.Println("tolerance:", geom.Epsilon[int](), geom.Epsilon[float32](), geom.Epsilon[float64]())
	for _, overshoot := range []float64{5e-7, 5e-6} {
		point := geom.Pt(10+overshoot, 5.0)
		fmt.Printf("  %g past the edge: Contains %t, DistanceTo %g\n", overshoot, rectangle.Contains(point), rectangle.DistanceTo(point))
	}
}

func integers() {
	rectangle := geom.RectangleFromMinMax(geom.Pt(0, 0), geom.Pt(10, 10))

	fmt.Println("integer rectangle 0..10:")
	fmt.Println("  contains (10,10):", rectangle.Contains(geom.Pt(10, 10)))
	fmt.Println("  contains (11,10):", rectangle.Contains(geom.Pt(11, 10)))
}

func cost() {
	fmt.Println("what each answer costs:")
	for _, m := range []struct {
		name  string
		bench func(*testing.B)
	}{
		{"EqualDelta", equalDelta},
		{"Equal", equal},
		{"EqualRelative", equalRelative},
		{"Rectangle.IntersectsRectangle", rectangleIntersectsRectangle},
		{"Segment.IntersectsRectangle, apart", segmentApart},
		{"Segment.IntersectsRectangle, through", segmentThrough},
		{"Polygon.Contains, outside its extent", polygonOutside},
		{"Polygon.Contains, inside", polygonInside},
	} {
		result := testing.Benchmark(m.bench)
		fmt.Printf("  %-38s %6.2f ns, %d allocations\n", m.name, float64(result.T.Nanoseconds())/float64(result.N), result.AllocsPerOp())
	}
}

func equalDelta(bench *testing.B) {
	for i := range bench.N {
		sinkBool = geom.EqualDelta(float64(i), float64(i), geom.Delta)
	}
}

func equal(bench *testing.B) {
	for i := range bench.N {
		sinkBool = geom.Equal(float64(i), float64(i))
	}
}

func equalRelative(bench *testing.B) {
	for i := range bench.N {
		sinkBool = geom.EqualRelative(float64(i), float64(i))
	}
}

func rectangleIntersectsRectangle(bench *testing.B) {
	rectangle := geom.Rect(geom.Pt(0.0, 0.0), geom.Sz(10.0, 10.0), 0)
	other := geom.Rect(geom.Pt(5.0, 5.0), geom.Sz(10.0, 10.0), 0)
	for bench.Loop() {
		sinkBool = rectangle.IntersectsRectangle(other)
	}
}

func segmentApart(bench *testing.B) {
	rectangle := geom.Rect(geom.Pt(0.0, 0.0), geom.Sz(10.0, 10.0), 0)
	segment := geom.Seg(geom.Pt(-20.0, 20.0), geom.Pt(20.0, 20.0))
	for bench.Loop() {
		sinkBool = segment.IntersectsRectangle(rectangle)
	}
}

func segmentThrough(bench *testing.B) {
	rectangle := geom.Rect(geom.Pt(0.0, 0.0), geom.Sz(10.0, 10.0), 0)
	segment := geom.Seg(geom.Pt(-20.0, 0.0), geom.Pt(20.0, 0.0))
	for bench.Loop() {
		sinkBool = segment.IntersectsRectangle(rectangle)
	}
}

func polygonOutside(bench *testing.B) {
	polygon := benchPolygon()
	point := geom.Pt(500.0, 500.0)
	for bench.Loop() {
		sinkBool = polygon.Contains(point)
	}
}

func polygonInside(bench *testing.B) {
	polygon := benchPolygon()
	point := geom.Pt(10.0, 20.0)
	for bench.Loop() {
		sinkBool = polygon.Contains(point)
	}
}

func benchPolygon() geom.Polygon[float64] {
	return geom.RegPol(geom.Pt(0.0, 0.0), geom.SzU(100.0), 64, 0, 0).Polygon()
}
