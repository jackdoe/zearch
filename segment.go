package main

import (
	"bytes"
	"fmt"
	mmap "github.com/edsrzf/mmap-go"
	"os"
	"sort"
	"sync"
	"unsafe"
)

type MMaped struct {
	fd *os.File
	m  mmap.MMap
}

func NewMMaped(name string) *MMaped {
	p := &MMaped{
		fd: openOrPanic(name),
	}
	var err error
	p.m, err = mmap.MapRegion(p.fd, -1, mmap.RDONLY, 0, 0)
	if err != nil {
		p.m = mmap.MMap{}
	}
	return p
}

func (m *MMaped) close() {
	m.m.Unmap()
	m.fd.Close()
}

func (m *MMaped) seekToStart() {
	m.fd.Seek(0, 0)
}

func (m *MMaped) writeOrPanic(b []byte) {
	writeOrPanic(m.fd, b)
}

type StoredStringArray struct {
	data   *MMaped
	header *MMaped
}

func NewStoredStringArray(name string) *StoredStringArray {
	s := &StoredStringArray{
		data:   NewMMaped(fmt.Sprintf("%s.data", name)),
		header: NewMMaped(fmt.Sprintf("%s.header", name)),
	}

	return s
}

func (s *StoredStringArray) close() {
	s.data.close()
	s.header.close()
}

func (s *StoredStringArray) write(input []string, cb func(string) uint64) {
	b8 := make([]byte, 8)
	off := uint32(0)
	s.header.seekToStart()
	s.data.seekToStart()

	for i := 0; i < len(input); i++ {
		bstr := []byte(input[i])
		putUint64(b8, uint64(off)<<32|uint64(len(bstr)))
		s.header.writeOrPanic(b8)

		extra := cb(input[i])
		putUint64(b8, extra)
		s.header.writeOrPanic(b8)

		s.data.writeOrPanic(bstr)
		off += uint32(len(bstr))
	}
}

func (s *StoredStringArray) count() int {
	return len(s.header.m) / 16
}

func (s *StoredStringArray) bcmp(offa, lena uint32, b []byte) int {
	return bytes.Compare(s.data.m[offa:offa+lena], b)
}

func (s *StoredStringArray) read(id uint32) (string, bool) {
	size := s.count()
	if id > uint32(size) {
		return "", false
	}
	offlen := getUint64(s.header.m, uint32(id*16))
	off := uint32(offlen >> 32)
	len := uint32(offlen & 0xFFFFFFFF)

	return string(s.data.m[off : off+len]), true
}

func (s *StoredStringArray) bsearch(input []byte) (uint64, bool) {
	start := 0
	end := s.count()
	for start < end {
		mid := start + ((end - start) / 2)
		offlen := getUint64(s.header.m, uint32(mid*16))

		offa := uint32(offlen >> 32)
		lena := uint32(offlen & 0xFFFFFFFF)
		diff := s.bcmp(offa, lena, input)

		if diff == 0 {
			extra := getUint64(s.header.m, uint32(mid*16)+8)
			return extra, true
		}
		if diff < 0 {
			start = mid + 1
		} else {
			end = mid
		}
	}
	return 0, false
}

type Segment struct {
	inmemoryInverted map[string][]int32
	inmemoryForward  []string
	inverted         *StoredStringArray
	forward          *StoredStringArray
	postings         *MMaped
	name             string
	sync.Mutex
}

func NewSegment(name string) *Segment {
	return &Segment{
		inmemoryInverted: make(map[string][]int32),
		inmemoryForward:  make([]string, 100),
		inverted:         NewStoredStringArray(fmt.Sprintf("%s.inverted", name)),
		forward:          NewStoredStringArray(fmt.Sprintf("%s.forward", name)),
		postings:         NewMMaped(fmt.Sprintf("%s.postings", name)),
		name:             name,
	}
}
func (s *Segment) close() {
	s.inverted.close()
	s.forward.close()
	s.postings.close()
}
func (s *Segment) findPostingsList(term string) []byte {
	extra, ok := s.inverted.bsearch([]byte(term))
	if ok {
		off := uint32(extra >> 32)
		l := uint32(extra & 0xFFFFFFFF)
		return s.postings.m[int(off):int(off+l)]
	}
	return []byte{}
}

func (s *Segment) addForward(doc string) int32 {
	id := len(s.inmemoryForward)
	s.inmemoryForward = append(s.inmemoryForward, doc)
	return int32(id)
}

func (s *Segment) addInverted(term string, id int32) {
	s.inmemoryInverted[term] = append(s.inmemoryInverted[term], id)
}

func unsafeCompare(a string, b string) int {
	abp := *(*[]byte)(unsafe.Pointer(&a))
	bbp := *(*[]byte)(unsafe.Pointer(&b))
	return bytes.Compare(abp, bbp)
}

type ByBytes []string

func (s ByBytes) Len() int {
	return len(s)
}
func (s ByBytes) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByBytes) Less(i, j int) bool {
	return unsafeCompare(s[i], s[j]) < 0
}

func (s *Segment) flushToDisk() {
	terms := make([]string, len(s.inmemoryInverted))
	i := 0
	for k := range s.inmemoryInverted {
		terms[i] = k
		i++
	}
	sort.Sort(ByBytes(terms))

	postings_off := int64(0)
	s.postings.seekToStart()
	s.inverted.write(terms, func(st string) uint64 {
		tpostings := s.inmemoryInverted[st]
		plen := len(tpostings) * 4
		ret := uint64(postings_off)<<32 | uint64(plen)
		buf := make([]byte, plen)
		postings_off += int64(plen)

		soff := 0
		for _, id := range tpostings {
			putUint32Off(buf, soff, uint32(id))
			soff += 4
		}
		s.postings.writeOrPanic(buf)
		return ret
	})

	s.forward.write(s.inmemoryForward, func(st string) uint64 {
		return uint64(0)
	})
}
