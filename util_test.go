package main

import "testing"

type TokenizerTestSeq struct {
	input      string
	s_expected []string
	w_expected []int
}

var tokenizerTests = []TokenizerTestSeq{
	{
		"hello",
		[]string{"hello"},
		[]int{0},
	},
	{
		"public static class AtomicLong foobar",
		[]string{"public", "static", "class", "AtomicLong", "foobar"},
		[]int{0, 0, 0, 1, 1},
	},
	{
		"0xdeadbeef hello sub hello_world {\npanic picnic\nsub main",
		[]string{"0xdeadbeef", "hello", "sub", "hello_world", "panic", "picnic", "sub", "main"},
		[]int{0, 0, 0, 1, 0, 0, 0, 1},
	},
}

func eq_int(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func eq_string(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func TestTokenizer(t *testing.T) {
	for _, tt := range tokenizerTests {
		s_actual := []string{}
		w_actual := []int{}
		tokenize(tt.input, func(s string, w int) {
			w_actual = append(w_actual, w)
			s_actual = append(s_actual, s)
		})
		if !eq_string(s_actual, tt.s_expected) {
			t.Errorf("expected %#v, actual %#v", tt.s_expected, s_actual)
		}

		if !eq_int(w_actual, tt.w_expected) {
			t.Errorf("expected %#v, actual %#v", tt.w_expected, w_actual)
		}
	}
}
