package main

import (
	"io"
	"log"
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
			cb(text)
		}
	}
}

func took(name string, r func()) {
	start := time.Now()
	r()
	elapsed := time.Since(start)
	log.Printf("%s: %fs", name, elapsed.Seconds())
}
