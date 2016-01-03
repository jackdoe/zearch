package main

import (
	lz4 "github.com/bkaradzic/go-lz4"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"sync"
)

type Segment struct {
	data     *StoredSegment
	inmemory map[string][]int32
	sync.Mutex
}

func NewSegment() *Segment {
	return &Segment{
		data:     &StoredSegment{},
		inmemory: make(map[string][]int32),
	}
}

func (s *Segment) findPostingsList(term string) []int32 {
	if p, ok := s.inmemory[term]; ok {
		return p
	}
	return []int32{}
}

// functions used only during indexing
func (s *Segment) addForward(doc string) int32 {
	id := len(s.data.Documents)
	s.data.Documents = append(s.data.Documents, doc)
	return int32(id)
}

func (s *Segment) addInverted(term string, id int32) {
	s.inmemory[term] = append(s.inmemory[term], id)
}

func (d *Segment) flushToDisk(path string) {
	// sort the terms

	data, err := proto.Marshal(d.data)
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

func (d *Segment) loadFromDisk(path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	data, err = lz4.Decode(nil, data)
	if err != nil {
		panic(err)
	}

	err = proto.Unmarshal(data, d.data)
	if err != nil {
		panic(err)
	}

}
