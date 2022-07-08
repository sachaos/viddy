package view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strconv"
	"sync"
)

type Record struct {
	ID int64
}

type HistoryView struct {
	*tview.Table
	display chan int
	sync.Mutex
	index map[int64]*HistoryRow
}

func NewHistoryView() *HistoryView {
	display := make(chan int, 10)

	table := tview.NewTable()

	v := &HistoryView{
		Table: table,
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

	h.index[r.ID] = hr
}

func (h *HistoryView) Finish(r *Record) {
	h.Lock()
	defer h.Unlock()

	hr := h.index[r.ID]
	hr.Finish()
}

type HistoryRow struct {
	id       *tview.TableCell
	addition *tview.TableCell
	deletion *tview.TableCell
	exitCode *tview.TableCell
}

func NewHistoryRecord(r *Record) *HistoryRow {
	idCell := tview.NewTableCell(strconv.FormatInt(r.ID, 10)).SetTextColor(tview.Styles.SecondaryTextColor)
	additionCell := tview.NewTableCell("").SetTextColor(tcell.ColorGreen)
	deletionCell := tview.NewTableCell("").SetTextColor(tcell.ColorRed)
	exitCodeCell := tview.NewTableCell("").SetTextColor(tcell.ColorYellow)

	return &HistoryRow{
		id:       idCell,
		addition: additionCell,
		deletion: deletionCell,
		exitCode: exitCodeCell,
	}
}

func (r *HistoryRow) Finish() {
	// TODO: fix for test
	r.addition.SetText("+" + strconv.Itoa(0))
	r.deletion.SetText("-" + strconv.Itoa(0))
}
