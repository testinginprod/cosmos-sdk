package collections

import (
	"strconv"
)

type int64Key struct {
}

func (i int64Key) Encode(key int64) []byte {
	b := make([]byte, 8)
	b[0] = byte(key >> 56)
	b[1] = byte(key >> 48)
	b[2] = byte(key >> 40)
	b[3] = byte(key >> 32)
	b[4] = byte(key >> 24)
	b[5] = byte(key >> 16)
	b[6] = byte(key >> 8)
	b[7] = byte(key)
	b[0] ^= 0x80
	return b
}

func (i int64Key) Decode(b []byte) (int, int64) {
	_ = b[7] // bounds check hint to compiler; see golang.org/issue/14808
	return 8, int64(b[7]) | int64(b[6])<<8 | int64(b[5])<<16 | int64(b[4])<<24 |
		int64(b[3])<<32 | int64(b[2])<<40 | int64(b[1])<<48 | int64(b[0]^0x80)<<56
}

func (i int64Key) Stringify(key int64) string {
	return strconv.FormatInt(key, 10)
}

var Int64KeyEncoder KeyEncoder[int64] = int64Key{}
