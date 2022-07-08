package store

import "github.com/sachaos/viddy/pkg/run"

type Record struct {
	ID int64
	Result *run.Result
}

type Store struct {
	records          []*Record
	index            map[int64]int
	currentSelection int
}

func NewStore() *Store {
	return &Store{
		records:          nil,
		index:            map[int64]int{},
	}
}

func (h *Store) Set(r *Record) {
	i, ok := h.index[r.ID]
	if !ok {
		h.records = append(h.records, r)
		h.index[r.ID] = len(h.records) - 1
		return
	}

	h.records[i] = r
}

func (h *Store) GetByIndex(i int) *Record {
	if h.Count() <= i || i < 0 {
		return nil
	}

	return h.records[i]
}

func (h *Store) GetIndexOf(r *Record) int {
	i, ok := h.index[r.ID]
	if !ok {
		return -1
	}
	return i
}

func (h *Store) Count() int {
	return len(h.records)
}
