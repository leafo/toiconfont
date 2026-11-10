// Package svgpath parses SVG path data and converts it to quadratic outlines.
package svgpath

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Point struct{ X, Y float64 }

type Op uint8

const (
	MoveTo Op = iota
	LineTo
	QuadTo
	CubicTo
	Close
)

// Cmd is one absolute path command. Pts holds the control points followed by
// the end point: 1 for MoveTo and LineTo, 2 for QuadTo, 3 for CubicTo.
type Cmd struct {
	Op  Op
	Pts [3]Point
}

type scanner struct {
	s   string
	i   int
	err error
}

func (sc *scanner) fail(msg string) {
	if sc.err == nil {
		sc.err = fmt.Errorf("svgpath: %s at offset %d", msg, sc.i)
	}
}

func (sc *scanner) skip() {
	for sc.i < len(sc.s) {
		switch sc.s[sc.i] {
		case ' ', '\t', '\n', '\r', ',':
			sc.i++
		default:
			return
		}
	}
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func (sc *scanner) digits() int {
	n := 0
	for sc.i < len(sc.s) && isDigit(sc.s[sc.i]) {
		sc.i++
		n++
	}
	return n
}

func (sc *scanner) number() float64 {
	sc.skip()
	if sc.err != nil {
		return 0
	}
	start := sc.i
	if sc.i < len(sc.s) && (sc.s[sc.i] == '+' || sc.s[sc.i] == '-') {
		sc.i++
	}
	n := sc.digits()
	if sc.i < len(sc.s) && sc.s[sc.i] == '.' {
		sc.i++
		n += sc.digits()
	}
	if n == 0 {
		sc.fail("expected number")
		return 0
	}
	if sc.i < len(sc.s) && (sc.s[sc.i] == 'e' || sc.s[sc.i] == 'E') {
		j := sc.i + 1
		if j < len(sc.s) && (sc.s[j] == '+' || sc.s[j] == '-') {
			j++
		}
		if j < len(sc.s) && isDigit(sc.s[j]) {
			sc.i = j
			sc.digits()
		}
	}
	v, err := strconv.ParseFloat(sc.s[start:sc.i], 64)
	if err != nil {
		sc.fail("invalid number")
	}
	return v
}

func (sc *scanner) flag() bool {
	sc.skip()
	if sc.err != nil {
		return false
	}
	if sc.i < len(sc.s) && (sc.s[sc.i] == '0' || sc.s[sc.i] == '1') {
		sc.i++
		return sc.s[sc.i-1] == '1'
	}
	sc.fail("expected arc flag")
	return false
}

func reflect(ctrl, around Point) Point {
	return Point{2*around.X - ctrl.X, 2*around.Y - ctrl.Y}
}

// Parse converts SVG path data into absolute commands. Arcs are returned as
// cubic curves.
func Parse(d string) ([]Cmd, error) {
	sc := &scanner{s: d}
	var out []Cmd
	var cmd, prev byte
	var cur, start, ctrl Point

	for {
		sc.skip()
		if sc.i >= len(sc.s) {
			return out, nil
		}
		if c := sc.s[sc.i]; strings.IndexByte("MmLlHhVvCcSsQqTtAaZz", c) >= 0 {
			cmd = c
			sc.i++
		} else {
			switch cmd {
			case 0, 'Z', 'z':
				sc.fail(fmt.Sprintf("unexpected %q", c))
				return nil, sc.err
			case 'M':
				cmd = 'L'
			case 'm':
				cmd = 'l'
			}
		}

		rel := cmd >= 'a'
		up := cmd &^ 0x20
		origin := cur
		point := func() Point {
			p := Point{sc.number(), sc.number()}
			if rel {
				p.X += origin.X
				p.Y += origin.Y
			}
			return p
		}

		switch up {
		case 'M':
			cur = point()
			start = cur
			out = append(out, Cmd{Op: MoveTo, Pts: [3]Point{cur}})
		case 'L':
			cur = point()
			out = append(out, Cmd{Op: LineTo, Pts: [3]Point{cur}})
		case 'H':
			x := sc.number()
			if rel {
				x += origin.X
			}
			cur = Point{x, origin.Y}
			out = append(out, Cmd{Op: LineTo, Pts: [3]Point{cur}})
		case 'V':
			y := sc.number()
			if rel {
				y += origin.Y
			}
			cur = Point{origin.X, y}
			out = append(out, Cmd{Op: LineTo, Pts: [3]Point{cur}})
		case 'C', 'S':
			c1 := origin
			if up == 'C' {
				c1 = point()
			} else if prev == 'C' || prev == 'S' {
				c1 = reflect(ctrl, origin)
			}
			ctrl = point()
			cur = point()
			out = append(out, Cmd{Op: CubicTo, Pts: [3]Point{c1, ctrl, cur}})
		case 'Q', 'T':
			c := origin
			if up == 'Q' {
				c = point()
			} else if prev == 'Q' || prev == 'T' {
				c = reflect(ctrl, origin)
			}
			ctrl = c
			cur = point()
			out = append(out, Cmd{Op: QuadTo, Pts: [3]Point{ctrl, cur}})
		case 'A':
			rx, ry, rot := sc.number(), sc.number(), sc.number()
			large, sweep := sc.flag(), sc.flag()
			cur = point()
			if sc.err == nil {
				out = append(out, arc(origin, rx, ry, rot, large, sweep, cur)...)
			}
		case 'Z':
			cur = start
			out = append(out, Cmd{Op: Close})
		}

		if sc.err != nil {
			return nil, sc.err
		}
		prev = up
	}
}

// arc converts an SVG elliptical arc to cubic curves, following the endpoint
// to center conversion from the SVG implementation notes.
func arc(from Point, rx, ry, rotation float64, large, sweep bool, to Point) []Cmd {
	if from == to {
		return nil
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	if rx == 0 || ry == 0 {
		return []Cmd{{Op: LineTo, Pts: [3]Point{to}}}
	}

	sin, cos := math.Sincos(rotation * math.Pi / 180)
	dx, dy := (from.X-to.X)/2, (from.Y-to.Y)/2
	x1, y1 := cos*dx+sin*dy, -sin*dx+cos*dy

	if l := x1*x1/(rx*rx) + y1*y1/(ry*ry); l > 1 {
		s := math.Sqrt(l)
		rx *= s
		ry *= s
	}

	num := rx*rx*ry*ry - rx*rx*y1*y1 - ry*ry*x1*x1
	den := rx*rx*y1*y1 + ry*ry*x1*x1
	coef := math.Sqrt(math.Max(0, num/den))
	if large == sweep {
		coef = -coef
	}
	cxp, cyp := coef*rx*y1/ry, -coef*ry*x1/rx
	cx := cos*cxp - sin*cyp + (from.X+to.X)/2
	cy := sin*cxp + cos*cyp + (from.Y+to.Y)/2

	theta := math.Atan2((y1-cyp)/ry, (x1-cxp)/rx)
	delta := math.Atan2((-y1-cyp)/ry, (-x1-cxp)/rx) - theta
	if sweep && delta < 0 {
		delta += 2 * math.Pi
	} else if !sweep && delta > 0 {
		delta -= 2 * math.Pi
	}

	n := int(math.Ceil(math.Abs(delta) / (math.Pi / 2)))
	if n < 1 {
		n = 1
	}
	step := delta / float64(n)
	t := 4.0 / 3.0 * math.Tan(step/4)

	at := func(a float64) (p, d Point) {
		s, c := math.Sincos(a)
		p = Point{cx + rx*c*cos - ry*s*sin, cy + rx*c*sin + ry*s*cos}
		d = Point{-rx*s*cos - ry*c*sin, -rx*s*sin + ry*c*cos}
		return
	}

	out := make([]Cmd, 0, n)
	p1, d1 := at(theta)
	for i := 1; i <= n; i++ {
		p2, d2 := at(theta + step*float64(i))
		if i == n {
			p2 = to
		}
		out = append(out, Cmd{Op: CubicTo, Pts: [3]Point{
			{p1.X + t*d1.X, p1.Y + t*d1.Y},
			{p2.X - t*d2.X, p2.Y - t*d2.Y},
			p2,
		}})
		p1, d1 = p2, d2
	}
	return out
}
