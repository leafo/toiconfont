package main

import (
	"math"
	"testing"
)

func TestIconDescent(t *testing.T) {
	// offsets measured from a font exported by IcoMoon, 1024 em, 6.25% baseline
	for _, c := range []struct{ grid, want float64 }{
		{0, 64},
		{16, 64},
		{32, 64},
		{14, 64 + 9.14},
		{20, 64 - 12.8},
		{24, 64 + 21.33},
	} {
		if got := iconDescent(64, 1024, c.grid); math.Abs(got-c.want) > 0.01 {
			t.Errorf("grid %v: got %.2f, want %.2f", c.grid, got, c.want)
		}
	}
}
