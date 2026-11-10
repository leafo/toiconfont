// Package woff2 wraps sfnt tables in a WOFF 2.0 container.
package woff2

import (
	"encoding/binary"

	"github.com/itchio/go-brotli/enc"
	"github.com/leafo/toiconfont/ttf"
)

// index in this list is the table's flag value in the directory
var knownTags = []string{"cmap", "head", "hhea", "hmtx", "maxp", "name", "OS/2", "post", "cvt ", "fpgm", "glyf", "loca", "prep"}

const arbitraryTag = 63

func base128(b []byte, v uint32) []byte {
	for shift := 28; shift > 0; shift -= 7 {
		if v>>shift != 0 {
			b = append(b, byte(v>>shift&0x7f)|0x80)
		}
	}
	return append(b, byte(v&0x7f))
}

// Encode expects tables sorted by tag, as returned by ttf.Build.
func Encode(tables []ttf.Table) ([]byte, error) {
	const headerSize = 48

	var dir, body []byte
	sfntSize := 12 + 16*len(tables)

	for _, t := range tables {
		flag := byte(arbitraryTag)
		for i, tag := range knownTags {
			if tag == t.Tag {
				flag = byte(i)
			}
		}
		// glyf and loca are stored as is, which for them is transform version 3
		if t.Tag == "glyf" || t.Tag == "loca" {
			flag |= 3 << 6
		}
		dir = append(dir, flag)
		if flag == arbitraryTag {
			dir = append(dir, t.Tag...)
		}
		dir = base128(dir, uint32(len(t.Data)))

		body = append(body, t.Data...)
		sfntSize += (len(t.Data) + 3) &^ 3
	}

	compressed, err := enc.CompressBuffer(body, &enc.BrotliWriterOptions{Quality: 11})
	if err != nil {
		return nil, err
	}

	be := binary.BigEndian
	out := make([]byte, headerSize, headerSize+len(dir)+len(compressed)+3)
	out = append(out, dir...)
	out = append(out, compressed...)
	for len(out)%4 != 0 {
		out = append(out, 0)
	}

	copy(out, "wOF2")
	be.PutUint32(out[4:], 0x00010000)
	be.PutUint32(out[8:], uint32(len(out)))
	be.PutUint16(out[12:], uint16(len(tables)))
	be.PutUint32(out[16:], uint32(sfntSize))
	be.PutUint32(out[20:], uint32(len(compressed)))
	be.PutUint16(out[24:], 1)

	return out, nil
}
