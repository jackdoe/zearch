package main

import (
	"bufio"
	lz4 "github.com/bkaradzic/go-lz4"
	msgpack "gopkg.in/vmihailenco/msgpack.v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"text/scanner"
)

var ONLY = map[string]bool{
	".java": true,
	".c":    true,
	".go":   true,
}

type Index struct {
	Inverted map[string][]int32
	Forward  []string
	sync.Mutex
}

func NewIndex() *Index {
	return &Index{
		Inverted: map[string][]int32{},
		Forward:  []string{},
	}
}

func (d *Index) postingList(token string) []int32 {
	if val, ok := d.Inverted[token]; ok {
		return val
	}
	return []int32{}
}

func (d *Index) load(path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	data, err = lz4.Decode(nil, data)
	if err != nil {
		panic(err)
	}

	if err = msgpack.Unmarshal(data, d); err != nil {
		panic(err)
	}
}

func (d *Index) dumpToDisk(path string) {
	data, err := msgpack.Marshal(d)
	if err != nil {
		panic(err)
	}

	data, err = lz4.Encode(nil, data)
	if err != nil {
		panic(err)
	}

	if err = ioutil.WriteFile(path, data, 0644); err != nil {
		panic(err)
	}
}

func (d *Index) adder(input chan string, done chan int) {
	sca := &scanner.Scanner{}
	uniq := map[string]int{}
	inc := func(text string, n int) {
		if len(text) > 0 {
			if current, ok := uniq[text]; ok {
				n += current
			}
			uniq[text] = n
		}
	}
	edge := func(text string, max int) {
		for i := 2; i < len(text); i++ {
			// create left edge ngrams with increasing weight
			// at
			// ato
			// atom
			// atomic
			// ..
			inc(text[:i+1], max/(len(text)-i))
		}
	}

	for {
		select {
		case path := <-input:
			f, err := os.Open(path)
			if err != nil {
				continue
			}

			initScanner(sca, bufio.NewReader(f))

			tokenize(sca, func(text string) {
				if len(text) > 2 {
					inc(text, 1)
				}
			})

			dir, name := filepath.Split(path)
			for _, di := range strings.Split(strings.ToLower(dir), "/") {
				inc(di, 1)
			}
			ext := filepath.Ext(name)
			name = strings.ToLower(strings.TrimSuffix(name, ext))
			edge(name, FILENAME_WEIGHT)
			inc(ext[1:], FILENAME_WEIGHT)

			d.Lock()

			id := int32(len(d.Forward))
			d.Forward = append(d.Forward, path)

			for text, count := range uniq {
				if count > 1024 {
					count = 1024
				}

				d.Inverted[text] = append(d.Inverted[text], id<<10|int32(count))
			}
			d.Unlock()

			f.Close()
			for k := range uniq {
				delete(uniq, k)
			}
		case <-done:
			return
		}
	}
}
func (d *Index) doIndex(args []string) {
	log.Printf("%#v\n", args)
	done := make(chan int)
	workers := make(chan string)
	maxproc := runtime.GOMAXPROCS(0)
	for i := 0; i < maxproc; i++ {
		log.Printf("starting indexer: %d/%d", i, maxproc)
		go func() {
			d.adder(workers, done)
		}()
	}
	walker := func(path string, f os.FileInfo, err error) error {
		if f != nil {
			name := f.Name()
			if !strings.HasPrefix(name, ".") && !f.IsDir() {
				ext := filepath.Ext(name)
				if _, ok := ONLY[ext]; ok {
					workers <- path
				}
			}
		}
		return nil
	}
	for _, arg := range args {
		if err := filepath.Walk(arg, walker); err != nil {
			panic(err)
		}
	}
	done <- 1
	close(workers)
	close(done)
}
