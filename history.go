package main

import (
	"context"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strconv"
	"sync"
)

type History struct {
	in       chan *Record
	finished chan *Record

	store *HistoryStore
	view  *HistoryView
}

func NewHistory() *History {
	in := make(chan *Record, 10)
	finished := make(chan *Record, 10)

	return &History{
		in:       in,
		finished: finished,
		store:    nil,
		view:     nil,
	}
}

func (h *History) Run(ctx context.Context) error {
	for {
		select {
		case r := <-h.in:
			h.store.Append(r)
			h.view.Append(r)
		case r := <-h.finished:
			h.view.Finish(r)
		case <-ctx.Done():
			return nil
		}
	}
}

func (h *History) View() *HistoryView {
	return h.view
}

func (h *History) In() chan<- *Record {
	return h.in
}

func (h *History) Finished() chan<- *Record {
	return h.finished
}

type Record struct {
	timestamp int64
}

type HistoryStore struct {
	records          []*Record
	index            map[int64]int
	currentSelection int
}

func (h *HistoryStore) Append(r *Record) {
	h.records = append(h.records, r)
}

func (h *HistoryStore) GetByIndex(i int) *Record {
	if h.Count() <= i || i < 0 {
		return nil
	}

	return h.records[i]
}

func (h *HistoryStore) GetIndexOf(r *Record) int {
	i, ok := h.index[r.timestamp]
	if !ok {
		return -1
	}
	return i
}

func (h *HistoryStore) Count() int {
	return len(h.records)
}

func (h *HistoryStore) GetSelection() int {
	return h.currentSelection
}

type HistoryView struct {
	*tview.Table
	display chan int
	sync.Mutex
	index map[int64]*HistoryRecord
}

func NewHistoryView() *HistoryView {
	t := tview.NewTable()

	display := make(chan int, 10)

	v := &HistoryView{
		Table:   t,
		display: display,
	}
	v.SetTitle("History")
	v.SetTitleAlign(tview.AlignLeft)
	v.SetBorder(true)
	v.ScrollToBeginning()
	v.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorGray))
	v.SetSelectionChangedFunc(func(row, column int) {
		display <- row
	})

	return v
}

func (h *HistoryView) Display() <-chan int {
	return h.display
}

func (h *HistoryView) Append(r *Record) {
	hr := NewHistoryRecord(r)

	h.Lock()
	defer h.Unlock()

	h.InsertRow(0)
	h.SetCell(0, 0, hr.id)
	h.SetCell(0, 1, hr.addition)
	h.SetCell(0, 2, hr.deletion)
	h.SetCell(0, 3, hr.exitCode)

	h.index[r.timestamp] = hr
}

func (h *HistoryView) Finish(r *Record) {
	h.Lock()
	defer h.Unlock()

	hr := h.index[r.timestamp]
	hr.Finish()
}

type HistoryRecord struct {
	id       *tview.TableCell
	addition *tview.TableCell
	deletion *tview.TableCell
	exitCode *tview.TableCell
}

func NewHistoryRecord(r *Record) *HistoryRecord {
	idCell := tview.NewTableCell(strconv.FormatInt(r.timestamp, 10)).SetTextColor(tview.Styles.SecondaryTextColor)
	additionCell := tview.NewTableCell("").SetTextColor(tcell.ColorGreen)
	deletionCell := tview.NewTableCell("").SetTextColor(tcell.ColorRed)
	exitCodeCell := tview.NewTableCell("").SetTextColor(tcell.ColorYellow)

	return &HistoryRecord{
		id:       idCell,
		addition: additionCell,
		deletion: deletionCell,
		exitCode: exitCodeCell,
	}
}

func (r *HistoryRecord) Finish() {
	r.addition.SetText("+" + strconv.Itoa(s.diffAdditionCount))
	r.deletion.SetText("-" + strconv.Itoa(s.diffDeletionCount))
}
