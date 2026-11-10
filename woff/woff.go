// Package woff wraps sfnt tables in a WOFF 1.0 container.
package woff

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"

	"github.com/leafo/toiconfont/ttf"
)

// Encode expects tables sorted by tag, as returned by ttf.Build.
func Encode(tables []ttf.Table) ([]byte, error) {
	const headerSize, entrySize = 44, 20

	be := binary.BigEndian
	dir := make([]byte, headerSize+entrySize*len(tables))
	var body []byte
	sfntSize := 12 + 16*len(tables)

	for i, t := range tables {
		var z bytes.Buffer
		zw, err := zlib.NewWriterLevel(&z, zlib.BestCompression)
		if err != nil {
			return nil, err
		}
		if _, err := zw.Write(t.Data); err != nil {
			return nil, err
		}
		if err := zw.Close(); err != nil {
			return nil, err
		}
		data := t.Data
		if z.Len() < len(t.Data) {
			data = z.Bytes()
		}

		e := dir[headerSize+entrySize*i:]
		copy(e, t.Tag)
		be.PutUint32(e[4:], uint32(len(dir)+len(body)))
		be.PutUint32(e[8:], uint32(len(data)))
		be.PutUint32(e[12:], uint32(len(t.Data)))
		be.PutUint32(e[16:], t.Sum)

		body = append(body, data...)
		for len(body)%4 != 0 {
			body = append(body, 0)
		}
		sfntSize += (len(t.Data) + 3) &^ 3
	}

	copy(dir, "wOFF")
	be.PutUint32(dir[4:], 0x00010000)
	be.PutUint32(dir[8:], uint32(len(dir)+len(body)))
	be.PutUint16(dir[12:], uint16(len(tables)))
	be.PutUint32(dir[16:], uint32(sfntSize))
	be.PutUint16(dir[20:], 1)

	return append(dir, body...), nil
}
