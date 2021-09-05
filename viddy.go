package main

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type HistoryRow struct {
	id *tview.TableCell

	addition *tview.TableCell
	deletion *tview.TableCell
}

type Viddy struct {
	begin int64

	keymap keymapping

	cmd  string
	args []string

	duration  time.Duration
	snapshots sync.Map

	intervalView *tview.TextView
	commandView  *tview.TextView
	timeView     *tview.TextView
	historyView  *tview.Table
	historyRows  map[int64]*HistoryRow
	sync.RWMutex

	idList []int64

	bodyView    *tview.TextView
	app         *tview.Application
	logView     *tview.TextView
	statusView  *tview.TextView
	queryEditor *tview.InputField

	snapshotQueue <-chan *Snapshot
	queue         chan int64
	finishedQueue chan int64
	diffQueue     chan int64

	currentID        int64
	latestFinishedID int64
	isTimeMachine    bool
	isSuspend        bool
	isNoTitle        bool
	isShowDiff       bool
	isEditQuery      bool

	query string

	isDebug     bool
	showLogView bool
}

type ViddyIntervalMode string

var (
	ViddyIntervalModeClockwork  ViddyIntervalMode = "clockwork"
	ViddyIntervalModePrecise    ViddyIntervalMode = "precise"
	ViddyIntervalModeSequential ViddyIntervalMode = "sequential"

	errCannotCreateSnapshot = errors.New("cannot find the snapshot")
	errNotCompletedYet      = errors.New("not completed yet")
)

func NewViddy(conf *config) *Viddy {
	begin := time.Now().UnixNano()

	newSnap := func(id int64, before *Snapshot, finish chan<- struct{}) *Snapshot {
		return NewSnapshot(id, conf.runtime.cmd, conf.runtime.args, conf.general.shell, conf.general.shellOptions, before, finish)
	}

	var snapshotQueue <-chan *Snapshot

	switch conf.runtime.mode {
	case ViddyIntervalModeClockwork:
		snapshotQueue = ClockSnapshot(begin, newSnap, conf.runtime.interval)
	case ViddyIntervalModeSequential:
		snapshotQueue = SequentialSnapshot(newSnap, conf.runtime.interval)
	case ViddyIntervalModePrecise:
		snapshotQueue = PreciseSnapshot(newSnap, conf.runtime.interval)
	}

	return &Viddy{
		keymap: conf.keymap,

		begin:       begin,
		cmd:         conf.runtime.cmd,
		args:        conf.runtime.args,
		duration:    conf.runtime.interval,
		snapshots:   sync.Map{},
		historyRows: map[int64]*HistoryRow{},

		snapshotQueue: snapshotQueue,
		queue:         make(chan int64),
		finishedQueue: make(chan int64),
		diffQueue:     make(chan int64, 100),

		isShowDiff: conf.runtime.differences,
		isNoTitle:  conf.runtime.noTitle,
		isDebug:    conf.general.debug,

		currentID:        -1,
		latestFinishedID: -1,
	}
}

func (v *Viddy) ShowLogView(b bool) {
	v.showLogView = b
	v.arrange()
}

func (v *Viddy) SetIsNoTitle(b bool) {
	v.isNoTitle = b
	v.arrange()
}

func (v *Viddy) SetIsShowDiff(b bool) {
	v.isShowDiff = b
	v.setSelection(v.currentID)
	v.arrange()
}

func (v *Viddy) SetIsTimeMachine(b bool) {
	v.isTimeMachine = b
	if !v.isTimeMachine {
		v.setSelection(v.latestFinishedID)
	}

	v.arrange()
}

func (v *Viddy) println(a ...interface{}) {
	fmt.Fprintln(v.logView, a...)
}

func (v *Viddy) addSnapshot(s *Snapshot) {
	v.snapshots.Store(s.id, s)
}

func (v *Viddy) startRunner() {
	for s := range v.snapshotQueue {
		v.addSnapshot(s)
		v.queue <- s.id

		_ = s.run(v.finishedQueue)
	}
}

func (v *Viddy) diffQueueHandler() {
	for {
		func() {
			defer v.app.Draw()

			id := <-v.diffQueue
			s := v.getSnapShot(id)

			if s == nil {
				return
			}

			err := s.compareFromBefore()
			if err != nil {
				time.Sleep(1 * time.Second)
				v.diffQueue <- id

				return
			}

			r, ok := v.historyRows[id]
			if !ok {
				return
			}

			r.addition.SetText("+" + strconv.Itoa(s.diffAdditionCount))
			r.deletion.SetText("-" + strconv.Itoa(s.diffDeletionCount))
		}()
	}
}

//nolint:funlen,gocognit,cyclop
func (v *Viddy) queueHandler() {
	for {
		func() {
			defer v.app.Draw()

			select {
			case id := <-v.finishedQueue:
				r, ok := v.historyRows[id]
				if !ok {
					return
				}

				r.id.SetTextColor(tview.Styles.PrimaryTextColor)

				s := v.getSnapShot(id)
				if s == nil {
					return
				}

				v.diffQueue <- s.id

				ls := v.getSnapShot(v.latestFinishedID)
				if ls == nil || s.start.After(ls.start) {
					v.latestFinishedID = id
					if !v.isTimeMachine {
						v.setSelection(id)
					} else {
						v.setSelection(v.currentID)
					}
				}
			case id := <-v.queue:
				if v.isSuspend {
					return
				}

				s := v.getSnapShot(id)
				idCell := tview.NewTableCell(strconv.FormatInt(s.id, 10)).SetTextColor(tview.Styles.SecondaryTextColor)
				additionCell := tview.NewTableCell("+0").SetTextColor(tcell.ColorGreen)
				deletionCell := tview.NewTableCell("-0").SetTextColor(tcell.ColorRed)

				v.historyRows[s.id] = &HistoryRow{
					id:       idCell,
					addition: additionCell,
					deletion: deletionCell,
				}

				v.historyView.InsertRow(0)
				v.historyView.SetCell(0, 0, idCell)
				v.historyView.SetCell(0, 1, additionCell)
				v.historyView.SetCell(0, 2, deletionCell)

				v.Lock()
				v.idList = append(v.idList, id)
				v.Unlock()

				if !v.isTimeMachine {
					v.setSelection(v.latestFinishedID)
				} else {
					v.setSelection(v.currentID)
				}
			}
		}()
	}
}

func (v *Viddy) setSelection(id int64) {
	if id == -1 {
		return
	}

	v.historyView.ScrollToBeginning()

	isSelectable, _ := v.historyView.GetSelectable()
	if !isSelectable {
		v.historyView.SetSelectable(true, false)
	}

	v.RLock()
	index := sort.Search(len(v.idList), func(i int) bool {
		return v.idList[i] >= id
	})
	i := len(v.idList) - index - 1
	v.RUnlock()

	v.historyView.Select(i, 0)
	v.currentID = id
	unix := v.begin + id*int64(time.Millisecond)
	v.timeView.SetText(time.Unix(unix/int64(time.Second), unix%int64(time.Second)).String())
}

func (v *Viddy) getSnapShot(id int64) *Snapshot {
	s, ok := v.snapshots.Load(id)
	if !ok {
		return nil
	}

	return s.(*Snapshot)
}

func (v *Viddy) renderSnapshot(id int64) error {
	s := v.getSnapShot(id)
	if s == nil {
		return errCannotCreateSnapshot
	}

	if !s.completed {
		v.bodyView.Clear()

		return errNotCompletedYet
	}

	v.bodyView.Clear()

	return s.render(v.bodyView, v.isShowDiff, v.query)
}

func (v *Viddy) UpdateStatusView() {
	v.statusView.SetText(fmt.Sprintf("Time Machine: %s  Suspend: %s  Diff: %s",
		convertToOnOrOff(v.isTimeMachine), convertToOnOrOff(v.isSuspend), convertToOnOrOff(v.isShowDiff)))
}

func convertToOnOrOff(on bool) string {
	if on {
		return "[green]ON [reset]"
	}

	return "[red]OFF[reset]"
}

func (v *Viddy) arrange() {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	if !v.isNoTitle {
		flex.AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(v.intervalView, 10, 1, false).
				AddItem(v.commandView, 0, 1, false).
				AddItem(v.statusView, 45, 1, false).
				AddItem(v.timeView, 21, 1, false),
			3, 1, false)
	}

	body := tview.NewFlex().SetDirection(tview.FlexRow)
	body.AddItem(v.bodyView, 0, 1, false)

	if v.isEditQuery || v.query != "" {
		body.AddItem(v.queryEditor, 1, 1, false)
	}

	middle := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(body, 0, 1, false)

	if v.isTimeMachine {
		middle.AddItem(v.historyView, 21, 1, true)
	}

	flex.AddItem(
		middle,
		0, 1, false)

	if v.showLogView {
		flex.AddItem(v.logView, 10, 1, false)
	}

	v.app.SetRoot(flex, true)
}

//nolint: funlen,gocognit,cyclop
func (v *Viddy) Run() error {
	b := tview.NewTextView()
	b.SetDynamicColors(true)
	b.SetTitle("body")
	b.SetRegions(true)
	v.bodyView = b

	t := tview.NewTextView()
	t.SetBorder(true).SetTitle("Time")
	v.timeView = t

	h := tview.NewTable()
	h.SetBorder(true).SetTitle("History")
	h.ScrollToBeginning()
	h.SetSelectionChangedFunc(func(row, column int) {
		c := h.GetCell(row, column)
		id, err := strconv.ParseInt(c.Text, 10, 64)
		if err == nil {
			_ = v.renderSnapshot(id)
		}
	})
	h.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorGray))

	v.historyView = h

	var cmd []string
	cmd = append(cmd, v.cmd)
	cmd = append(cmd, v.args...)

	c := tview.NewTextView()
	c.SetBorder(true).SetTitle("Command")
	c.SetText(strings.Join(cmd, " "))
	v.commandView = c

	d := tview.NewTextView()
	d.SetBorder(true).SetTitle("Every")
	d.SetText(v.duration.String())
	v.intervalView = d

	s := tview.NewTextView()
	s.SetBorder(true).SetTitle("Status")
	s.SetDynamicColors(true)
	v.statusView = s

	l := tview.NewTextView()
	l.SetBorder(true).SetTitle("Log")
	l.ScrollToEnd()
	v.logView = l

	q := tview.NewInputField().SetLabel("/")
	q.SetChangedFunc(func(text string) {
		v.query = text
	})
	q.SetDoneFunc(func(key tcell.Key) {
		v.isEditQuery = false
		v.arrange()
	})

	v.queryEditor = q

	app := tview.NewApplication()
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		v.println(fmt.Sprintf("key: %+v", event))

		if v.isEditQuery {
			v.queryEditor.InputHandler()(event, nil)

			return event
		}

		keystroke := KeyStroke{
			Key:     event.Key(),
			Rune:    event.Rune(),
			ModMask: event.Modifiers(),
		}

		var any bool
		if _, ok := v.keymap.toggleTimeMachine[keystroke]; ok {
			v.SetIsTimeMachine(!v.isTimeMachine)
			any = true
		}

		if _, ok := v.keymap.goToPastOnTimeMachine[keystroke]; ok {
			if !v.isTimeMachine {
				return event
			}
			v.goToPastOnTimeMachine()
			any = true
		}

		if _, ok := v.keymap.goToFutureOnTimeMachine[keystroke]; ok {
			if !v.isTimeMachine {
				return event
			}
			v.goToFutureOnTimeMachine()
			any = true
		}

		if _, ok := v.keymap.goToMorePastOnTimeMachine[keystroke]; ok {
			if !v.isTimeMachine {
				return event
			}
			v.goToMorePastOnTimeMachine()
			any = true
		}

		if _, ok := v.keymap.goToMoreFutureOnTimeMachine[keystroke]; ok {
			if !v.isTimeMachine {
				return event
			}
			v.goToMoreFutureOnTimeMachine()
			any = true
		}

		if _, ok := v.keymap.goToNowOnTimeMachine[keystroke]; ok {
			if !v.isTimeMachine {
				return event
			}
			v.goToNowOnTimeMachine()
			any = true
		}

		if _, ok := v.keymap.goToOldestOnTimeMachine[keystroke]; ok {
			if !v.isTimeMachine {
				return event
			}
			v.goToOldestOnTimeMachine()
			any = true
		}

		switch event.Rune() {
		case 's':
			v.isSuspend = !v.isSuspend
		case 'd':
			v.SetIsShowDiff(!v.isShowDiff)
		case 't':
			v.SetIsNoTitle(!v.isNoTitle)
		case 'x':
			if v.isDebug {
				v.ShowLogView(!v.showLogView)
			}
		case '/':
			if v.query != "" {
				v.query = ""
				v.queryEditor.SetText("")
			}
			v.isEditQuery = true
			v.arrange()
		default:
			if !any {
				v.bodyView.InputHandler()(event, nil)
			}
		}

		v.UpdateStatusView()

		return event
	})

	v.app = app

	go v.diffQueueHandler()
	go v.queueHandler()
	go v.startRunner()

	v.UpdateStatusView()

	app.EnableMouse(true)

	v.arrange()

	return app.Run()
}

func (v *Viddy) goToPastOnTimeMachine() {
	count := v.historyView.GetRowCount()
	selection, _ := v.historyView.GetSelection()

	if selection+1 < count {
		cell := v.historyView.GetCell(selection+1, 0)
		if id, err := strconv.ParseInt(cell.Text, 10, 64); err == nil {
			v.setSelection(id)
		}
	}
}

func (v *Viddy) goToFutureOnTimeMachine() {
	selection, _ := v.historyView.GetSelection()
	if 0 <= selection-1 {
		cell := v.historyView.GetCell(selection-1, 0)
		if id, err := strconv.ParseInt(cell.Text, 10, 64); err == nil {
			v.setSelection(id)
		}
	}
}

func (v *Viddy) goToMorePastOnTimeMachine() {
	count := v.historyView.GetRowCount()
	selection, _ := v.historyView.GetSelection()

	if selection+10 < count {
		cell := v.historyView.GetCell(selection+10, 0)
		if id, err := strconv.ParseInt(cell.Text, 10, 64); err == nil {
			v.setSelection(id)
		}
	} else {
		cell := v.historyView.GetCell(count-1, 0)
		if id, err := strconv.ParseInt(cell.Text, 10, 64); err == nil {
			v.setSelection(id)
		}
	}
}

func (v *Viddy) goToMoreFutureOnTimeMachine() {
	selection, _ := v.historyView.GetSelection()
	if 0 <= selection-10 {
		cell := v.historyView.GetCell(selection-10, 0)
		if id, err := strconv.ParseInt(cell.Text, 10, 64); err == nil {
			v.setSelection(id)
		}
	} else {
		cell := v.historyView.GetCell(0, 0)
		if id, err := strconv.ParseInt(cell.Text, 10, 64); err == nil {
			v.setSelection(id)
		}
	}
}

func (v *Viddy) goToNowOnTimeMachine() {
	cell := v.historyView.GetCell(0, 0)
	if id, err := strconv.ParseInt(cell.Text, 10, 64); err == nil {
		v.setSelection(id)
	}
}

func (v *Viddy) goToOldestOnTimeMachine() {
	count := v.historyView.GetRowCount()
	cell := v.historyView.GetCell(count-1, 0)

	if id, err := strconv.ParseInt(cell.Text, 10, 64); err == nil {
		v.setSelection(id)
	}
}
