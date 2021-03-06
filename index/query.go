package index

import (
	"math"
	"sort"
)

const (
	NO_MORE   = int32(math.MaxInt32)
	NOT_READY = int32(-1)
)

type Query interface {
	advance(int32) int32
	Next() int32
	GetDocId() int32
	AddSubQuery(Query)
	Score() int64
	Cost() uint32
	Prepare(*Segment)
}

type QueryBase struct {
	docId int32
}

func (q *QueryBase) GetDocId() int32 {
	return q.docId
}

type Term struct {
	cursor   int32
	postings []byte
	term     string
	QueryBase
}

func (t *Term) AddSubQuery(q Query) {
	// noop
}

func (t *Term) Prepare(s *Segment) {
	t.cursor = 0
	t.docId = NOT_READY
	t.postings = s.findPostingsList(t.term)
}

func (t *Term) Cost() uint32 {
	return uint32(len(t.postings) / 4)
}

func (t *Term) Score() int64 {
	return int64(1) + int64(t.getAt(t.cursor)&0x3FF)
}

func NewTerm(term string) *Term {
	return &Term{
		cursor:    0,
		term:      term,
		QueryBase: QueryBase{NOT_READY},
	}
}

func (t *Term) getAt(idx int32) uint32 {
	return getUint32(t.postings, uint32(idx*4))
}
func (t *Term) advance(target int32) int32 {
	if t.docId == NO_MORE || t.docId == target || target == NO_MORE {
		t.docId = target
		return t.docId
	}
	start := t.cursor
	end := int32(len(t.postings) / 4)
	for start < end {
		mid := start + ((end - start) / 2)
		current := int32(t.getAt(mid) >> 10)
		if current == target {
			t.cursor = mid
			t.docId = target
			return t.GetDocId()
		}

		if current < target {
			start = mid + 1
		} else {
			end = mid
		}
	}

	return t.move(start)
}

func (t *Term) move(to int32) int32 {
	t.cursor = to
	if t.cursor >= int32(len(t.postings)/4) {
		t.docId = NO_MORE
	} else {
		t.docId = int32(t.getAt(t.cursor) >> 10)
	}
	return t.docId
}

func (t *Term) Next() int32 {
	if t.docId != NOT_READY {
		t.cursor++
	}
	return t.move(t.cursor)
}

type BoolQueryBase struct {
	queries []Query
}

func (q *BoolQueryBase) Prepare(s *Segment) {
	for i := 0; i < len(q.queries); i++ {
		q.queries[i].Prepare(s)
	}
	sort.Sort(ByCost(q.queries))
}

type ByCost []Query

func (s ByCost) Len() int {
	return len(s)
}
func (s ByCost) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByCost) Less(i, j int) bool {
	return s[i].Cost() < s[j].Cost()
}

func (q *BoolQueryBase) AddSubQuery(sub Query) {
	q.queries = append(q.queries, sub)
}

type BoolOrQuery struct {
	BoolQueryBase
	QueryBase
}

func NewBoolOrQuery(queries []Query) *BoolOrQuery {
	return &BoolOrQuery{
		BoolQueryBase: BoolQueryBase{queries},
		QueryBase:     QueryBase{NOT_READY},
	}
}

func (q *BoolOrQuery) Cost() uint32 {
	sum := uint32(0)
	for i := 0; i < len(q.queries); i++ {
		sum += q.queries[i].Cost()
	}
	return sum
}

func (q *BoolOrQuery) Score() int64 {
	total := int64(0)
	for i := 0; i < len(q.queries); i++ {
		if q.queries[i].GetDocId() == q.GetDocId() {
			total += q.queries[i].Score()
		}
	}
	return total
}

func (q *BoolOrQuery) advance(target int32) int32 {
	new_doc := NO_MORE
	for _, sub_query := range q.queries {
		cur_doc := sub_query.GetDocId()
		if cur_doc < target {
			cur_doc = sub_query.advance(target)
		}

		if cur_doc < new_doc {
			new_doc = cur_doc
		}
	}
	q.docId = new_doc
	return q.docId
}

func (q *BoolOrQuery) Next() int32 {
	new_doc := NO_MORE
	for _, sub_query := range q.queries {
		cur_doc := sub_query.GetDocId()
		if cur_doc == q.docId {
			cur_doc = sub_query.Next()
		}

		if cur_doc < new_doc {
			new_doc = cur_doc
		}
	}
	q.docId = new_doc
	return new_doc
}

type BoolAndQuery struct {
	BoolQueryBase
	QueryBase
}

func NewBoolAndQuery(queries []Query) *BoolAndQuery {
	return &BoolAndQuery{
		BoolQueryBase: BoolQueryBase{queries},
		QueryBase:     QueryBase{NOT_READY},
	}
}

func (q *BoolAndQuery) Cost() uint32 {
	if len(q.queries) == 0 {
		return uint32(0)
	}

	min := uint32(math.MaxUint32)
	for i := 0; i < len(q.queries); i++ {
		cost := q.queries[i].Cost()
		if min > cost {
			min = cost
		}
	}
	return min
}

func (q *BoolAndQuery) Score() int64 {
	total := int64(0)
	for i := 0; i < len(q.queries); i++ {
		total += q.queries[i].Score()
	}
	return total
}

func (q *BoolAndQuery) nextAndedDoc(target int32) int32 {
	// initial iteration skips queries[0]
	for i := 1; i < len(q.queries); i++ {
		sub_query := q.queries[i]

		if sub_query.GetDocId() < target {
			sub_query.advance(target)
		}

		if sub_query.GetDocId() == target {
			continue
		}

		target = q.queries[0].advance(sub_query.GetDocId())
		i = 0 //restart the loop from the first query
	}
	q.docId = target
	return q.docId
}

func (q *BoolAndQuery) advance(target int32) int32 {
	if len(q.queries) == 0 {
		q.docId = NO_MORE
		return NO_MORE
	}

	return q.nextAndedDoc(q.queries[0].advance(target))
}

func (q *BoolAndQuery) Next() int32 {
	if len(q.queries) == 0 {
		q.docId = NO_MORE
		return NO_MORE
	}

	return q.nextAndedDoc(q.queries[0].Next())
}
