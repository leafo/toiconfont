package woff

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"testing"

	"github.com/leafo/toiconfont/ttf"
)

func TestBuildAndEncode(t *testing.T) {
	square := []ttf.Point{{X: 0, Y: 0, On: true}, {X: 0, Y: 800, On: true}, {X: 400, Y: 1000}, {X: 800, Y: 800, On: true}, {X: 800, Y: 0, On: true}}
	sfnt, tables, err := ttf.Build(ttf.Font{
		Family: "test", UnitsPerEm: 1024, Ascent: 960, Descent: 64,
		Glyphs: []ttf.Glyph{
			{Code: 0xe901, Advance: 1024, Contours: [][]ttf.Point{square}},
			{Code: 0xe900, Advance: 900, Contours: [][]ttf.Point{square}},
			{Code: 0x1f600, Advance: 1024, Contours: [][]ttf.Point{square}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if sum := ttf.Checksum(sfnt); sum != 0xB1B0AFBA {
		t.Errorf("file checksum is %08x", sum)
	}

	be := binary.BigEndian
	for i, table := range tables {
		rec := sfnt[12+16*i:]
		if string(rec[:4]) != table.Tag {
			t.Fatalf("table %d is %s, want %s", i, rec[:4], table.Tag)
		}
		offset, length := be.Uint32(rec[8:]), be.Uint32(rec[12:])
		if !bytes.Equal(sfnt[offset:offset+length], table.Data) {
			t.Errorf("%s: file data differs from table", table.Tag)
		}
	}

	out, err := Encode(tables)
	if err != nil {
		t.Fatal(err)
	}
	if int(be.Uint32(out[8:])) != len(out) || int(be.Uint32(out[16:])) != len(sfnt) {
		t.Errorf("header sizes are wrong")
	}
	for i, table := range tables {
		e := out[44+20*i:]
		offset, comp, orig := be.Uint32(e[4:]), be.Uint32(e[8:]), be.Uint32(e[12:])
		data := out[offset : offset+comp]
		if comp != orig {
			r, err := zlib.NewReader(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if data, err = io.ReadAll(r); err != nil {
				t.Fatal(err)
			}
		}
		if !bytes.Equal(data, table.Data) {
			t.Errorf("%s: woff data differs from table", table.Tag)
		}
	}
}
