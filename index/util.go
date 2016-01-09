package index

import (
	"log"
	"os"
	"time"
)

var WEIRD = map[string]int{
	"function": 1,
	"func":     1,
	"sub":      1,
	"class":    1,
}

func Tokenize(input string, cb func(string, int)) {
	weird := 0
	start, end := -1, -1
	for i, c := range input {
		if c == '\n' || c == '\r' {
			weird = 0
		}
		if c >= 'A' && c <= 'Z' {
			c |= 0x20
		}
		if (c >= 'a' && c <= 'z') || c == '_' || c == ':' || (c >= '0' && c <= '9') {
			if start == -1 {
				start = i
				end = start
			}
			end++
		} else {
			if end-start > 0 {
				s := input[start:end]
				if _, ok := WEIRD[s]; ok {
					weird = 1
					cb(s, 0)
				} else {
					cb(s, weird)
				}
			}
			start, end = -1, -1
		}
	}
	if end-start > 0 {
		cb(input[start:end], weird)
	}
}

func Took(name string, r func()) {
	start := time.Now()
	r()
	elapsed := time.Since(start)
	log.Printf("%s: %fs", name, elapsed.Seconds())
}

func getUint32(b []byte, off uint32) uint32 {
	return uint32(b[0+off]) | uint32(b[1+off])<<8 | uint32(b[2+off])<<16 | uint32(b[3+off])<<24
}

func putUint32Off(b []byte, off int, v uint32) {
	b[0+off] = byte(v)
	b[1+off] = byte(v >> 8)
	b[2+off] = byte(v >> 16)
	b[3+off] = byte(v >> 24)
}

func putUint32(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

func getUint64(b []byte, off uint32) uint64 {
	return uint64(b[0+off]) | uint64(b[1+off])<<8 | uint64(b[2+off])<<16 | uint64(b[3+off])<<24 |
		uint64(b[4+off])<<32 | uint64(b[5+off])<<40 | uint64(b[6+off])<<48 | uint64(b[7+off])<<56
}

func putUint64(b []byte, v uint64) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
	b[4] = byte(v >> 32)
	b[5] = byte(v >> 40)
	b[6] = byte(v >> 48)
	b[7] = byte(v >> 56)
}

func openOrPanic(name string) *os.File {
	fd, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	return fd
}

func writeOrPanic(fd *os.File, b []byte) {
	_, err := fd.Write(b)
	if err != nil {
		panic(err)
	}
}
