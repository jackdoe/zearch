package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/scanner"
)

var ONLY = map[string]bool{
	".java": true,
	".c":    true,
	".go":   true,
}

type Index struct {
	shards []*Segment
}

const N_SEGMENTS = 20

func NewIndex() *Index {
	s := make([]*Segment, N_SEGMENTS)
	for i := 0; i < N_SEGMENTS; i++ {
		s[i] = NewSegment()
	}
	return &Index{
		shards: s,
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
			s := d.shards[rand.Intn(len(d.shards))]
			s.Lock()
			id := s.addForward(path)

			for text, count := range uniq {
				if count > 1024 {
					count = 1024
				}
				s.addInverted(text, id<<10|int32(count))
			}
			s.Unlock()

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
	log.Printf("done")
}

func (d *Index) flushToDisk(path string) {
	for i, s := range d.shards {
		s.flushToDisk(fmt.Sprintf("%s.segment.%d", path, i))
	}
}

func (d *Index) loadFromDisk(path string) {
	for i, s := range d.shards {
		s.loadFromDisk(fmt.Sprintf("%s.segment.%d", path, i))
	}
}

func (d *Index) executeQuery(query Query, cb func(string, int32, int64)) {
	for i := 0; i < len(d.shards); i++ {
		query.Prepare(d.shards[i])
		for query.Next() != NO_MORE {
			id := query.GetDocId()
			cb(d.shards[i].data.Documents[id], int32(i)<<24|id, query.Score())
		}
	}
}

func (d *Index) fetchForward(id int) (string, bool) {
	shard := int(id >> 24)
	id = id & 0x00FFFFFF
	if shard < 0 || shard > len(d.shards) {
		return "", false
	}
	if id < 0 || id > len(d.shards[shard].data.Documents)-1 {
		return "", false
	}
	return d.shards[shard].data.Documents[id], true
}

func (d *Index) stats() (int, int) {
	total := 0
	approxterms := 0
	for _, s := range d.shards {
		total += len(s.data.Documents)
		approxterms += len(s.data.Postings)
	}
	return total, approxterms
}
