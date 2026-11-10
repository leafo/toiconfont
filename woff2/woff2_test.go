package woff2

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/itchio/dskompress/brotli"
	"github.com/leafo/toiconfont/ttf"
)

func readBase128(b []byte) (v uint32, n int) {
	for {
		v = v<<7 | uint32(b[n]&0x7f)
		n++
		if b[n-1]&0x80 == 0 {
			return v, n
		}
	}
}

func TestBase128(t *testing.T) {
	for _, v := range []uint32{0, 63, 127, 128, 16383, 16384, 1 << 28, 0xFFFFFFFF} {
		b := base128(nil, v)
		if got, n := readBase128(b); got != v || n != len(b) {
			t.Errorf("%d encoded as % x reads back as %d", v, b, got)
		}
	}
}

func TestEncode(t *testing.T) {
	square := []ttf.Point{{X: 0, Y: 0, On: true}, {X: 0, Y: 800, On: true}, {X: 400, Y: 1000}, {X: 800, Y: 800, On: true}, {X: 800, Y: 0, On: true}}
	sfnt, tables, err := ttf.Build(ttf.Font{
		Family: "test", UnitsPerEm: 1024, Ascent: 960, Descent: 64,
		Glyphs: []ttf.Glyph{
			{Code: 0xe900, Advance: 900, Contours: [][]ttf.Point{square}},
			{Code: 0xe901, Advance: 1024, Contours: [][]ttf.Point{square}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	tables = append(tables, ttf.Table{Tag: "zzzz", Data: bytes.Repeat([]byte{7}, 300)})
	sfntSize := len(sfnt) + 16 + 300

	out, err := Encode(tables)
	if err != nil {
		t.Fatal(err)
	}

	be := binary.BigEndian
	if string(out[:4]) != "wOF2" || int(be.Uint32(out[8:])) != len(out) || len(out)%4 != 0 {
		t.Fatalf("bad header")
	}
	if int(be.Uint16(out[12:])) != len(tables) || int(be.Uint32(out[16:])) != sfntSize {
		t.Errorf("header counts are wrong")
	}

	pos := 48
	var lengths []int
	for _, table := range tables {
		flag := out[pos]
		pos++
		tag := "zzzz"
		if flag&63 == arbitraryTag {
			pos += 4
		} else {
			tag = knownTags[flag&63]
		}
		if tag != table.Tag {
			t.Fatalf("directory has %s, want %s", tag, table.Tag)
		}
		if want := tag == "glyf" || tag == "loca"; (flag>>6 == 3) != want {
			t.Errorf("%s: transform version %d", tag, flag>>6)
		}
		length, n := readBase128(out[pos:])
		pos += n
		lengths = append(lengths, int(length))
	}

	compressed := out[pos : pos+int(be.Uint32(out[20:]))]
	r, err := brotli.NewReader(bytes.NewReader(compressed), nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	for i, table := range tables {
		if lengths[i] > len(body) || !bytes.Equal(body[:lengths[i]], table.Data) {
			t.Fatalf("%s: woff2 data differs from table", table.Tag)
		}
		body = body[lengths[i]:]
	}
	if len(body) != 0 {
		t.Errorf("%d trailing bytes", len(body))
	}
}
