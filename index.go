package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var ONLY = map[string]bool{
	".java": true,
	".c":    true,
	".go":   true,
	".pl":   true,
	".pm":   true,
}

type Index struct {
	shards []*Segment
}

func NewIndex(name string) *Index {
	s := fmt.Sprintf("%s/shard.*", name)
	matches, err := filepath.Glob(s)
	if err != nil {
		panic(err)
	}
	i := &Index{
		shards: []*Segment{},
	}
	if matches != nil {
		for _, match := range matches {
			i.shards = append(i.shards, NewSegment(match))
		}
	}

	return i
}

func (d *Index) executeQuery(query Query, cb func(int32, int64)) {
	for i := 0; i < len(d.shards); i++ {
		query.Prepare(d.shards[i])
		for query.Next() != NO_MORE {
			id := query.GetDocId()
			cb(int32(i)<<24|id, query.Score())
		}
	}
}

func (d *Index) fetchForward(id int) (string, bool) {
	shard := int(id >> 24)
	id = id & 0x00FFFFFF
	if shard < 0 || shard > len(d.shards) {
		return "", false
	}
	if s, ok := d.shards[shard].forward.read(uint32(id)); ok {
		return s, true
	}
	return "", false
}

func (d *Index) stats() (int, int) {
	total := 0
	approxterms := 0
	for _, s := range d.shards {
		total += s.forward.count()
		approxterms += s.inverted.count()
	}

	return total, approxterms
}

func (d *Index) close() {
	for _, s := range d.shards {
		s.close()
	}
}

type indexable struct {
	path    string
	segment *Segment
}

func tokenizeAndAdd(input chan indexable, done chan int) {
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
		case todo := <-input:
			data, err := ioutil.ReadFile(todo.path)
			if err != nil {
				log.Print(err)
				continue
			}
			tokenize(string(data), func(text string, weird int) {
				if len(text) > 2 {
					inc(text, 1+(weird*10))
				}
			})

			dir, name := filepath.Split(todo.path)
			for _, di := range strings.Split(dir, "/") {
				inc(di, 1)
			}
			ext := filepath.Ext(name)
			name = strings.TrimSuffix(name, ext)
			edge(name, FILENAME_WEIGHT)
			inc(ext[1:], FILENAME_WEIGHT)

			todo.segment.Lock()
			id := todo.segment.addForward(todo.path)

			for text, count := range uniq {
				if count > 1024 {
					count = 1024
				}
				todo.segment.addInverted(text, id<<10|int32(count))
			}

			todo.segment.Unlock()

			for k := range uniq {
				delete(uniq, k)
			}
		case <-done:
			return
		}
	}
}

func doIndex(name string, args []string) {
	log.Printf("%#v\n", args)

	maxproc := runtime.GOMAXPROCS(0)

	done := make(chan int)
	workers := make(chan indexable)

	inprogress := []*Segment{}
	n := 0
	current_n_segment := 0
	stop := func() {
		for i := 0; i < maxproc; i++ {
			done <- 1
		}
	}

	start := func() {
		for i := 0; i < maxproc; i++ {
			go func() {
				tokenizeAndAdd(workers, done)
			}()
		}
	}

	move := func(onlyflush bool) {
		stop()

		flushers := make(chan int)
		for _, s := range inprogress {
			go func(seg *Segment) {
				seg.flushToDisk()
				seg.close()
				flushers <- 1
			}(s)
		}

		for range inprogress {
			<-flushers
		}
		close(flushers)

		if !onlyflush {
			inprogress = []*Segment{}
			for i := current_n_segment; i < current_n_segment+4; i++ {
				s := fmt.Sprintf("%s/shard.%d", name, i)
				if err := os.MkdirAll(s, 0755); err != nil {
					panic(err)
				}

				log.Printf("creating new shard: %s", s)
				inprogress = append(inprogress, NewSegment(s))

			}
			current_n_segment += 4
		}
		runtime.GC()
		start()
	}

	start()
	move(false)
	walker := func(path string, f os.FileInfo, err error) error {
		if f != nil {
			name := f.Name()
			if !strings.HasPrefix(name, ".") && !f.IsDir() {
				ext := filepath.Ext(name)
				if _, ok := ONLY[ext]; ok {
					n++
					if n > 15000 {
						move(false)
						n = 0
					}

					workers <- indexable{path, inprogress[rand.Intn(len(inprogress))]}
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
	move(true)
	stop()
	close(workers)
	close(done)

	log.Printf("done")
}
