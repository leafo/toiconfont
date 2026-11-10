// Package ttf writes TrueType fonts made of simple glyphs.
package ttf

import (
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"unicode/utf16"
)

type Point struct {
	X, Y int16
	On   bool
}

type Glyph struct {
	Code     rune
	Advance  uint16
	Contours [][]Point
}

type Font struct {
	Family     string
	UnitsPerEm uint16
	Ascent     int16
	Descent    int16 // positive distance below the baseline
	Glyphs     []Glyph
}

// Table is one sfnt table. Data is not padded.
type Table struct {
	Tag  string
	Data []byte
	Sum  uint32
}

// seconds from 1904-01-01 to 2020-01-01, fixed so output is reproducible
const timestamp = 3660681600

type buf struct{ b []byte }

func (w *buf) u8(v uint8)   { w.b = append(w.b, v) }
func (w *buf) u16(v uint16) { w.b = binary.BigEndian.AppendUint16(w.b, v) }
func (w *buf) i16(v int16)  { w.u16(uint16(v)) }
func (w *buf) u32(v uint32) { w.b = binary.BigEndian.AppendUint32(w.b, v) }
func (w *buf) u64(v uint64) { w.b = binary.BigEndian.AppendUint64(w.b, v) }
func (w *buf) pad() {
	for len(w.b)%4 != 0 {
		w.b = append(w.b, 0)
	}
}

// Checksum is the sfnt table checksum of data.
func Checksum(data []byte) uint32 {
	var sum uint32
	for len(data) >= 4 {
		sum += binary.BigEndian.Uint32(data)
		data = data[4:]
	}
	if len(data) > 0 {
		var last [4]byte
		copy(last[:], data)
		sum += binary.BigEndian.Uint32(last[:])
	}
	return sum
}

func searchParams(n, unit int) (searchRange, entrySelector, rangeShift uint16) {
	p, e := 1, 0
	for p*2 <= n {
		p *= 2
		e++
	}
	return uint16(p * unit), uint16(e), uint16(n*unit - p*unit)
}

type bounds struct{ xMin, yMin, xMax, yMax int16 }

func (b *bounds) add(o bounds, first bool) {
	if first {
		*b = o
		return
	}
	b.xMin, b.yMin = min(b.xMin, o.xMin), min(b.yMin, o.yMin)
	b.xMax, b.yMax = max(b.xMax, o.xMax), max(b.yMax, o.yMax)
}

func encodeGlyph(g Glyph) (data []byte, bb bounds, points int, err error) {
	if len(g.Contours) == 0 {
		return nil, bb, 0, nil
	}

	var ends []uint16
	var flags, xs, ys []byte
	px, py := 0, 0

	coord := func(out []byte, delta int, short, same byte) ([]byte, byte, error) {
		switch {
		case delta == 0:
			return out, same, nil
		case delta > -256 && delta < 256:
			if delta > 0 {
				return append(out, byte(delta)), short | same, nil
			}
			return append(out, byte(-delta)), short, nil
		case delta < -32768 || delta > 32767:
			return nil, 0, fmt.Errorf("ttf: glyph %x has a coordinate step out of range", g.Code)
		}
		return binary.BigEndian.AppendUint16(out, uint16(int16(delta))), 0, nil
	}

	for _, contour := range g.Contours {
		if len(contour) == 0 {
			return nil, bb, 0, fmt.Errorf("ttf: glyph %x has an empty contour", g.Code)
		}
		for _, p := range contour {
			bb.add(bounds{p.X, p.Y, p.X, p.Y}, points == 0)

			var flag, f byte
			if p.On {
				flag |= 0x01
			}
			if points == 0 {
				flag |= 0x40 // contours may overlap
			}
			if xs, f, err = coord(xs, int(p.X)-px, 0x02, 0x10); err != nil {
				return nil, bb, 0, err
			}
			flag |= f
			if ys, f, err = coord(ys, int(p.Y)-py, 0x04, 0x20); err != nil {
				return nil, bb, 0, err
			}
			flag |= f

			flags = append(flags, flag)
			px, py = int(p.X), int(p.Y)
			points++
		}
		ends = append(ends, uint16(points-1))
	}
	if points > 65535 {
		return nil, bb, 0, fmt.Errorf("ttf: glyph %x has too many points", g.Code)
	}

	var w buf
	w.i16(int16(len(g.Contours)))
	w.i16(bb.xMin)
	w.i16(bb.yMin)
	w.i16(bb.xMax)
	w.i16(bb.yMax)
	for _, e := range ends {
		w.u16(e)
	}
	w.u16(0) // no instructions
	w.b = append(w.b, flags...)
	w.b = append(w.b, xs...)
	w.b = append(w.b, ys...)
	w.pad()
	return w.b, bb, points, nil
}

func cmapTable(glyphs []Glyph) []byte {
	type segment struct {
		start, end rune
		gid        int
	}
	var all, bmp []segment
	add := func(list []segment, code rune, gid int) []segment {
		if n := len(list); n > 0 && list[n-1].end+1 == code && list[n-1].gid+int(code-list[n-1].start) == gid {
			list[n-1].end = code
			return list
		}
		return append(list, segment{code, code, gid})
	}
	for i, g := range glyphs {
		if i == 0 {
			continue
		}
		all = add(all, g.Code, i)
		if g.Code < 0xFFFF {
			bmp = add(bmp, g.Code, i)
		}
	}
	full := len(all) > 0 && all[len(all)-1].end >= 0xFFFF

	var f4 buf
	n := len(bmp) + 1
	searchRange, entrySelector, rangeShift := searchParams(n, 2)
	f4.u16(4)
	f4.u16(uint16(16 + 8*n))
	f4.u16(0)
	f4.u16(uint16(n * 2))
	f4.u16(searchRange)
	f4.u16(entrySelector)
	f4.u16(rangeShift)
	for _, s := range bmp {
		f4.u16(uint16(s.end))
	}
	f4.u16(0xFFFF)
	f4.u16(0)
	for _, s := range bmp {
		f4.u16(uint16(s.start))
	}
	f4.u16(0xFFFF)
	for _, s := range bmp {
		f4.u16(uint16(s.gid - int(s.start)))
	}
	f4.u16(1)
	for i := 0; i < n; i++ {
		f4.u16(0)
	}

	var f12 buf
	if full {
		f12.u16(12)
		f12.u16(0)
		f12.u32(uint32(16 + 12*len(all)))
		f12.u32(0)
		f12.u32(uint32(len(all)))
		for _, s := range all {
			f12.u32(uint32(s.start))
			f12.u32(uint32(s.end))
			f12.u32(uint32(s.gid))
		}
	}

	type record struct {
		platform, encoding uint16
		format12           bool
	}
	records := []record{{0, 3, false}, {3, 1, false}}
	if full {
		records = []record{{0, 3, false}, {0, 4, true}, {3, 1, false}, {3, 10, true}}
	}

	var w buf
	w.u16(0)
	w.u16(uint16(len(records)))
	header := 4 + 8*len(records)
	for _, r := range records {
		w.u16(r.platform)
		w.u16(r.encoding)
		if r.format12 {
			w.u32(uint32(header + len(f4.b)))
		} else {
			w.u32(uint32(header))
		}
	}
	w.b = append(w.b, f4.b...)
	w.b = append(w.b, f12.b...)
	return w.b
}

func nameTable(family string) []byte {
	postscript := strings.Map(func(r rune) rune {
		if r <= ' ' || r > '~' || strings.ContainsRune("[](){}<>/%", r) {
			return -1
		}
		return r
	}, family)
	names := []struct {
		id    uint16
		value string
	}{
		{1, family},
		{2, "Regular"},
		{3, family + " 1.0"},
		{4, family},
		{5, "Version 1.0"},
		{6, postscript},
	}

	var w, strs buf
	w.u16(0)
	w.u16(uint16(len(names)))
	w.u16(uint16(6 + 12*len(names)))
	for _, n := range names {
		offset := len(strs.b)
		for _, u := range utf16.Encode([]rune(n.value)) {
			strs.u16(u)
		}
		w.u16(3)
		w.u16(1)
		w.u16(0x409)
		w.u16(n.id)
		w.u16(uint16(len(strs.b) - offset))
		w.u16(uint16(offset))
	}
	w.b = append(w.b, strs.b...)
	return w.b
}

// Build returns the font file and its tables sorted by tag.
func Build(f Font) ([]byte, []Table, error) {
	if len(f.Glyphs) == 0 {
		return nil, nil, fmt.Errorf("ttf: no glyphs")
	}
	if len(f.Glyphs) >= 65535 {
		return nil, nil, fmt.Errorf("ttf: too many glyphs")
	}
	if f.UnitsPerEm < 16 || f.UnitsPerEm > 16384 {
		return nil, nil, fmt.Errorf("ttf: units per em must be between 16 and 16384")
	}

	glyphs := make([]Glyph, 0, len(f.Glyphs)+1)
	glyphs = append(glyphs, Glyph{Advance: f.UnitsPerEm / 2})
	glyphs = append(glyphs, f.Glyphs...)
	sort.SliceStable(glyphs[1:], func(i, j int) bool { return glyphs[i+1].Code < glyphs[j+1].Code })
	for i := 1; i < len(glyphs); i++ {
		if c := glyphs[i].Code; c <= 0 || c > 0x10FFFF || c == 0xFFFF {
			return nil, nil, fmt.Errorf("ttf: invalid code %x", c)
		} else if c == glyphs[i-1].Code {
			return nil, nil, fmt.Errorf("ttf: duplicate code %x", c)
		}
	}

	var glyf, loca, hmtx buf
	var bb bounds
	var maxPoints, maxContours, drawn int
	var advanceMax uint16
	var advanceSum int
	var minLsb, minRsb, maxExtent int16

	for _, g := range glyphs {
		data, gb, points, err := encodeGlyph(g)
		if err != nil {
			return nil, nil, err
		}
		loca.u32(uint32(len(glyf.b)))
		glyf.b = append(glyf.b, data...)

		hmtx.u16(g.Advance)
		hmtx.i16(gb.xMin)
		advanceMax = max(advanceMax, g.Advance)
		advanceSum += int(g.Advance)

		if points == 0 {
			continue
		}
		maxPoints = max(maxPoints, points)
		maxContours = max(maxContours, len(g.Contours))
		rsb := int16(int(g.Advance) - int(gb.xMax))
		if drawn == 0 {
			minLsb, minRsb, maxExtent = gb.xMin, rsb, gb.xMax
		}
		minLsb, minRsb, maxExtent = min(minLsb, gb.xMin), min(minRsb, rsb), max(maxExtent, gb.xMax)
		bb.add(gb, drawn == 0)
		drawn++
	}
	loca.u32(uint32(len(glyf.b)))

	first, last := glyphs[1].Code, glyphs[len(glyphs)-1].Code
	em := int(f.UnitsPerEm)

	var head buf
	head.u32(0x00010000)
	head.u32(0x00010000)
	head.u32(0) // checkSumAdjustment, set once the file is assembled
	head.u32(0x5F0F3CF5)
	head.u16(0x0003)
	head.u16(f.UnitsPerEm)
	head.u64(timestamp)
	head.u64(timestamp)
	head.i16(bb.xMin)
	head.i16(bb.yMin)
	head.i16(bb.xMax)
	head.i16(bb.yMax)
	head.u16(0)
	head.u16(8)
	head.i16(2)
	head.i16(1) // long loca offsets
	head.i16(0)

	var hhea buf
	hhea.u32(0x00010000)
	hhea.i16(f.Ascent)
	hhea.i16(-f.Descent)
	hhea.i16(0)
	hhea.u16(advanceMax)
	hhea.i16(minLsb)
	hhea.i16(minRsb)
	hhea.i16(maxExtent)
	hhea.i16(1)
	hhea.i16(0)
	hhea.i16(0)
	hhea.u64(0)
	hhea.i16(0)
	hhea.u16(uint16(len(glyphs)))

	var maxp buf
	maxp.u32(0x00010000)
	maxp.u16(uint16(len(glyphs)))
	maxp.u16(uint16(maxPoints))
	maxp.u16(uint16(maxContours))
	maxp.u16(0)
	maxp.u16(0)
	maxp.u16(2)
	for i := 0; i < 8; i++ {
		maxp.u16(0)
	}

	var os2 buf
	os2.u16(4)
	os2.i16(int16(advanceSum / len(glyphs)))
	os2.u16(400)
	os2.u16(5)
	os2.u16(0)
	for _, v := range []int{em * 65 / 100, em * 70 / 100, 0, em * 14 / 100, em * 65 / 100, em * 70 / 100, 0, em * 48 / 100, em * 5 / 100, em * 26 / 100} {
		os2.i16(int16(v))
	}
	os2.i16(0)
	for i := 0; i < 10; i++ {
		os2.u8(0)
	}
	var range2 uint32
	for _, g := range glyphs[1:] {
		if g.Code >= 0xE000 && g.Code <= 0xF8FF {
			range2 |= 1 << 28 // private use area
		}
	}
	os2.u32(0)
	os2.u32(range2)
	os2.u32(0)
	os2.u32(0)
	os2.b = append(os2.b, "    "...)
	os2.u16(0x0040)
	os2.u16(uint16(min(first, 0xFFFF)))
	os2.u16(uint16(min(last, 0xFFFF)))
	os2.i16(f.Ascent)
	os2.i16(-f.Descent)
	os2.i16(0)
	os2.u16(uint16(max(f.Ascent, bb.yMax, 0)))
	os2.u16(uint16(max(f.Descent, -bb.yMin, 0)))
	os2.u32(1)
	os2.u32(0)
	os2.i16(0)
	os2.i16(0)
	os2.u16(0)
	os2.u16(32)
	os2.u16(0)

	var post buf
	post.u32(0x00030000)
	post.u32(0)
	post.i16(int16(-em / 10))
	post.i16(int16(em / 20))
	for i := 0; i < 5; i++ {
		post.u32(0)
	}

	tables := []Table{
		{Tag: "OS/2", Data: os2.b},
		{Tag: "cmap", Data: cmapTable(glyphs)},
		{Tag: "glyf", Data: glyf.b},
		{Tag: "head", Data: head.b},
		{Tag: "hhea", Data: hhea.b},
		{Tag: "hmtx", Data: hmtx.b},
		{Tag: "loca", Data: loca.b},
		{Tag: "maxp", Data: maxp.b},
		{Tag: "name", Data: nameTable(f.Family)},
		{Tag: "post", Data: post.b},
	}

	var file, body buf
	searchRange, entrySelector, rangeShift := searchParams(len(tables), 16)
	file.u32(0x00010000)
	file.u16(uint16(len(tables)))
	file.u16(searchRange)
	file.u16(entrySelector)
	file.u16(rangeShift)

	offset := 12 + 16*len(tables)
	headOffset := 0
	for i := range tables {
		t := &tables[i]
		t.Sum = Checksum(t.Data)
		if t.Tag == "head" {
			headOffset = offset + len(body.b)
		}
		file.b = append(file.b, t.Tag...)
		file.u32(t.Sum)
		file.u32(uint32(offset + len(body.b)))
		file.u32(uint32(len(t.Data)))
		body.b = append(body.b, t.Data...)
		body.pad()
	}
	file.b = append(file.b, body.b...)

	adjustment := 0xB1B0AFBA - Checksum(file.b)
	binary.BigEndian.PutUint32(file.b[headOffset+8:], adjustment)
	binary.BigEndian.PutUint32(head.b[8:], adjustment)

	return file.b, tables, nil
}
