package svgpath

import "math"

// OutlinePoint is a point of a quadratic outline. Off curve points are the
// control points of quadratic curves.
type OutlinePoint struct {
	Point
	On bool
}

// Outline converts commands into closed contours made of lines and quadratic
// curves. Cubic curves are approximated to within tolerance.
func Outline(cmds []Cmd, tolerance float64) [][]OutlinePoint {
	var out [][]OutlinePoint
	var cur []OutlinePoint
	var start, pos Point

	flush := func() {
		if len(cur) > 0 {
			out = append(out, cur)
		}
		cur = nil
	}
	begin := func() {
		if len(cur) == 0 {
			cur = append(cur, OutlinePoint{start, true})
			pos = start
		}
	}

	for _, c := range cmds {
		switch c.Op {
		case MoveTo:
			flush()
			start = c.Pts[0]
		case LineTo:
			begin()
			cur = append(cur, OutlinePoint{c.Pts[0], true})
			pos = c.Pts[0]
		case QuadTo:
			begin()
			cur = append(cur, OutlinePoint{c.Pts[0], false}, OutlinePoint{c.Pts[1], true})
			pos = c.Pts[1]
		case CubicTo:
			begin()
			cur = cubicToQuads(cur, pos, c.Pts[0], c.Pts[1], c.Pts[2], tolerance, 0)
			pos = c.Pts[2]
		case Close:
			flush()
		}
	}
	flush()
	return out
}

func mid(a, b Point) Point { return Point{(a.X + b.X) / 2, (a.Y + b.Y) / 2} }

// cubicToQuads appends quadratic curves approximating a cubic. A single
// quadratic with control point (3(p1+p2) - p0 - p3)/4 is at most
// sqrt(3)/36 * |p3 - 3p2 + 3p1 - p0| away from the cubic.
func cubicToQuads(out []OutlinePoint, p0, p1, p2, p3 Point, tolerance float64, depth int) []OutlinePoint {
	dx := p3.X - 3*p2.X + 3*p1.X - p0.X
	dy := p3.Y - 3*p2.Y + 3*p1.Y - p0.Y
	if depth >= 10 || math.Sqrt(3)/36*math.Hypot(dx, dy) <= tolerance {
		ctrl := Point{(3*(p1.X+p2.X) - p0.X - p3.X) / 4, (3*(p1.Y+p2.Y) - p0.Y - p3.Y) / 4}
		return append(out, OutlinePoint{ctrl, false}, OutlinePoint{p3, true})
	}

	a, b, c := mid(p0, p1), mid(p1, p2), mid(p2, p3)
	ab, bc := mid(a, b), mid(b, c)
	m := mid(ab, bc)
	out = cubicToQuads(out, p0, a, ab, m, tolerance, depth+1)
	return cubicToQuads(out, m, bc, c, p3, tolerance, depth+1)
}
