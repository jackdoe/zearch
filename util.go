package main

import (
	"io"
	"log"
	"os"
	"strings"
	"text/scanner"
	"time"
)

var SCANNERERROR = func(s *scanner.Scanner, msg string) {}

func initScanner(s *scanner.Scanner, src io.Reader) {
	s.Init(src)
	s.Error = SCANNERERROR
}

func isAlphaNumericNotOnlyDigit(s string) bool {
	nDigit := 0
	zerox := true
	for i, c := range s {
		if zerox && i == 1 && c == 'x' {
			return false
		}
		if c >= '0' && c <= '9' {
			nDigit++
			if i == 0 && c == '0' {
				zerox = true
			}
			continue
		}
		if (c < 'a' || c > 'z') && c != '_' {
			return false
		}
	}
	if nDigit == len(s) {
		return false
	}
	return true
}

func tokenize(s *scanner.Scanner, cb func(string)) {
	var tok rune
	for tok != scanner.EOF {
		tok = s.Scan()
		text := strings.ToLower(s.TokenText())
		if len(text) > 0 && isAlphaNumericNotOnlyDigit(text) {
			if len(text) > MAX_TOKEN_LEN {
				cb(text[:MAX_TOKEN_LEN])
			} else {
				cb(text)
			}
		}
	}
}

func took(name string, r func()) {
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
