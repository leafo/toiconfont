package svgpath

import (
	"math"
	"testing"
)

func TestParse(t *testing.T) {
	cmds, err := Parse("M10-5.5l5 5h-3V1e1q1 1 2 0t2 0c1,1 2,1 3,0s2-1 3 0a2 2 0 01-4 0z")
	if err != nil {
		t.Fatal(err)
	}

	var ends []Point
	for _, c := range cmds {
		switch c.Op {
		case MoveTo, LineTo:
			ends = append(ends, c.Pts[0])
		case QuadTo:
			ends = append(ends, c.Pts[1])
		case CubicTo:
			ends = append(ends, c.Pts[2])
		}
	}
	want := []Point{{10, -5.5}, {15, -0.5}, {12, -0.5}, {12, 10}, {14, 10}, {16, 10}, {19, 10}, {22, 10}}
	for i, w := range want {
		if ends[i] != w {
			t.Errorf("end %d: got %v, want %v", i, ends[i], w)
		}
	}
	if last := ends[len(ends)-1]; last != (Point{18, 10}) {
		t.Errorf("arc ends at %v", last)
	}
	if cmds[5].Pts[0] != (Point{15, 9}) {
		t.Errorf("smooth quad control is %v", cmds[5].Pts[0])
	}
	if cmds[len(cmds)-1].Op != Close {
		t.Error("missing close")
	}

	for _, bad := range []string{"10 10", "M10", "Mz", "M0 0a1 1 0 2 0 1 1"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}

func TestCubicToQuads(t *testing.T) {
	p := [4]Point{{0, 0}, {0, 600}, {400, 1000}, {1000, 1000}}
	const tolerance = 0.25
	contours := Outline([]Cmd{{Op: MoveTo, Pts: [3]Point{p[0]}}, {Op: CubicTo, Pts: [3]Point{p[1], p[2], p[3]}}}, tolerance)
	pts := contours[0]

	var samples []Point
	for i := 0; i+2 < len(pts); i += 2 {
		a, c, b := pts[i].Point, pts[i+1].Point, pts[i+2].Point
		for s := 0.0; s <= 1; s += 0.05 {
			samples = append(samples, Point{
				(1-s)*(1-s)*a.X + 2*s*(1-s)*c.X + s*s*b.X,
				(1-s)*(1-s)*a.Y + 2*s*(1-s)*c.Y + s*s*b.Y,
			})
		}
	}

	for _, q := range samples {
		best := math.Inf(1)
		for s := 0.0; s <= 1; s += 0.0005 {
			u := 1 - s
			x := u*u*u*p[0].X + 3*u*u*s*p[1].X + 3*u*s*s*p[2].X + s*s*s*p[3].X
			y := u*u*u*p[0].Y + 3*u*u*s*p[1].Y + 3*u*s*s*p[2].Y + s*s*s*p[3].Y
			best = math.Min(best, math.Hypot(x-q.X, y-q.Y))
		}
		if best > tolerance+0.3 {
			t.Fatalf("point %v is %.3f from the cubic", q, best)
		}
	}
}
