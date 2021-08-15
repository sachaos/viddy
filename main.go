package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Viddy struct {
	cmd  string
	args []string

	duration  time.Duration
	snapshots sync.Map

	intervalView *tview.TextView
	commandView  *tview.TextView
	timeView     *tview.TextView
	historyView  *tview.Table
	historyCells map[int64]*tview.TableCell
	sync.RWMutex

	idList []int64

	bodyView   *tview.TextView
	app        *tview.Application
	logView    *tview.TextView
	statusView *tview.TextView

	snapshotQueue <-chan *Snapshot
	queue         chan int64
	finishedQueue chan int64

	currentID        int64
	latestFinishedID int64
	isTimemachine    bool
	isSuspend        bool
	isNoTitle        bool
	isShowDiff       bool

	isDebug bool
}

func ClockSnapshot(name string, args []string, interval time.Duration) <-chan *Snapshot {
	c := make(chan *Snapshot)

	go func() {
		var s *Snapshot
		t := time.Tick(interval)

		for {
			select {
			case now := <-t:
				finish := make(chan struct{})
				id := now.UnixNano()
				s = NewSnapshot(id, name, args, s, finish)
				c <- s
			}
		}
	}()

	return c
}

func PreciseSnapshot(name string, args []string, interval time.Duration) <-chan *Snapshot {
	c := make(chan *Snapshot)

	go func() {
		var s *Snapshot

		for {
			finish := make(chan struct{})
			start := time.Now()
			id := start.UnixNano()
			ns := NewSnapshot(id, name, args, s, finish)
			s = ns
			c <- ns
			<-finish
			pTime := time.Since(start)

			if pTime > interval {
				continue
			} else {
				time.Sleep(interval - pTime)
			}
		}
	}()

	return c
}

func SequentialSnapshot(name string, args []string, interval time.Duration) <-chan *Snapshot {
	c := make(chan *Snapshot)

	go func() {
		var s *Snapshot

		for {
			finish := make(chan struct{})
			id := time.Now().UnixNano()
			s = NewSnapshot(id, name, args, s, finish)
			c <- s
			<-finish

			time.Sleep(interval)
		}
	}()

	return c
}

type ViddyIntervalMode string

var (
	ViddyIntervalModeActual     ViddyIntervalMode = "actual"
	ViddyIntervalModePrecise    ViddyIntervalMode = "precise"
	ViddyIntervalModeSequential ViddyIntervalMode = "sequential"
)

func NewViddy(duration time.Duration, cmd string, args []string, mode ViddyIntervalMode) *Viddy {
	var snapshotQueue <-chan *Snapshot
	switch mode {
	case ViddyIntervalModeActual:
		snapshotQueue = ClockSnapshot(cmd, args, duration)
	case ViddyIntervalModeSequential:
		snapshotQueue = SequentialSnapshot(cmd, args, duration)
	case ViddyIntervalModePrecise:
		snapshotQueue = PreciseSnapshot(cmd, args, duration)
	}

	return &Viddy{
		cmd:          cmd,
		args:         args,
		duration:     duration,
		snapshots:    sync.Map{},
		historyCells: map[int64]*tview.TableCell{},

		snapshotQueue: snapshotQueue,
		queue:         make(chan int64),
		finishedQueue: make(chan int64),
	}
}

func (v *Viddy) SetIsDebug(b bool) {
	v.isDebug = b
	v.arrange()
}

func (v *Viddy) SetIsNoTitle(b bool) {
	v.isNoTitle = b
	v.arrange()
}

func (v *Viddy) SetIsTimemachine(b bool) {
	v.isTimemachine = b
	if !v.isTimemachine {
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
	for {
		select {
		case s := <-v.snapshotQueue:
			v.addSnapshot(s)
			v.queue <- s.id

			s.run(v.finishedQueue)
		}
	}
}

func (v *Viddy) queueHandler() {
	for {
		func() {
			defer v.app.Draw()

			select {
			case id := <-v.finishedQueue:
				c, ok := v.historyCells[id]
				if !ok {
					return
				}
				c.SetTextColor(tcell.ColorWhite)

				s := v.getSnapShot(id)
				if s == nil {
					return
				}

				ls := v.getSnapShot(v.latestFinishedID)
				if ls == nil || s.start.After(ls.start) {
					v.latestFinishedID = id
					if !v.isTimemachine {
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
				c := tview.NewTableCell(strconv.FormatInt(s.id, 10))
				v.historyCells[s.id] = c

				c.SetTextColor(tcell.ColorDarkGray)

				v.historyView.InsertRow(0)
				v.historyView.SetCell(0, 0, c)

				v.Lock()
				v.idList = append(v.idList, id)
				v.Unlock()

				if !v.isTimemachine {
					v.setSelection(v.latestFinishedID)
				} else {
					v.setSelection(v.currentID)
				}
			}
		}()
	}
}

func (v *Viddy) setSelection(id int64) {
	if id == 0 {
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
	v.timeView.SetText(strconv.FormatInt(id, 10))
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
		return errors.New("cannot find the snapshot")
	}

	if !s.completed {
		v.bodyView.Clear()
		return errors.New("not completed yet")
	}

	v.println("render id:", id)
	v.bodyView.Clear()
	return s.render(v.bodyView, v.isShowDiff)
}

func (v *Viddy) UpdateStatusView() {
	v.statusView.SetText(fmt.Sprintf("Timemachine: %s  Suspend: %s  Diff: %s", convertToOnOrOff(v.isTimemachine), convertToOnOrOff(v.isSuspend), convertToOnOrOff(v.isShowDiff)))
}

func convertToOnOrOff(on bool) string {
	if on {
		return "[green]ON [white]"
	} else {
		return "[red]OFF[white]"
	}
}

func (v *Viddy) arrange() {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	if !v.isNoTitle {
		flex.AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(v.intervalView, 10, 1, false).
				AddItem(v.commandView, 0, 1, false).
				AddItem(v.statusView, 0, 1, false).
				AddItem(v.timeView, 20, 1, false),
			3, 1, false)
	}

	middle := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(v.bodyView, 0, 1, false)

	if v.isTimemachine {
		middle.AddItem(v.historyView, 20, 1, true)
	}

	flex.AddItem(
		middle,
		0, 1, false)

	if v.isDebug {
		flex.AddItem(v.logView, 10, 1, false)
	}

	v.app.SetRoot(flex, true)
}

func (v *Viddy) Run() error {
	b := tview.NewTextView()
	b.SetDynamicColors(true)
	b.SetTitle("body")
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
			v.renderSnapshot(id)
		}
	})
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

	app := tview.NewApplication()
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case ' ':
			v.SetIsTimemachine(!v.isTimemachine)
		case 's':
			v.isSuspend = !v.isSuspend
		case 'J':
			if !v.isTimemachine {
				return event
			}
			count := v.historyView.GetRowCount()
			selection, _ := v.historyView.GetSelection()

			if selection+1 < count {
				cell := v.historyView.GetCell(selection+1, 0)
				id, err := strconv.ParseInt(cell.Text, 10, 64)
				if err == nil {
					v.setSelection(id)
				}
			}
		case 'K':
			if !v.isTimemachine {
				return event
			}
			selection, _ := v.historyView.GetSelection()
			if 0 <= selection-1 {
				cell := v.historyView.GetCell(selection-1, 0)
				id, err := strconv.ParseInt(cell.Text, 10, 64)
				if err == nil {
					v.setSelection(id)
				}
			}
		case 'd':
			v.isShowDiff = !v.isShowDiff
		case 't':
			v.SetIsNoTitle(!v.isNoTitle)
		case 'x':
			v.SetIsDebug(!v.isDebug)
		default:
			v.bodyView.InputHandler()(event, nil)
		}

		v.UpdateStatusView()

		return event
	})
	v.app = app

	go v.queueHandler()
	go v.startRunner()

	v.UpdateStatusView()

	app.EnableMouse(true)

	v.arrange()

	if err := app.Run(); err != nil {
		return err
	}

	return nil
}

type Arguments struct {
	interval  time.Duration
	isPrecise bool
	isActual  bool
	isDebug   bool
	isDiff    bool
	isNoTitle bool

	cmd  string
	args []string
}

func parseArguments(args []string) (*Arguments, error) {
	argument := Arguments{
		interval: 2 * time.Second,
	}
	var err error

LOOP:
	for len(args) != 0 {
		arg := args[0]
		args = args[1:]

		switch arg {
		case "-n", "--interval":
			if len(args) == 0 {
				return nil, errors.New("-n or --interval require argument")
			}
			interval := args[0]
			args = args[1:]
			argument.interval, err = time.ParseDuration(interval)
			if err != nil {
				return nil, err
			}
		case "-p", "--precise":
			argument.isPrecise = true
		case "-a", "--actual":
			argument.isActual = true
		case "--debug":
			argument.isDebug = true
		case "-d", "--differences":
			argument.isDiff = true
		case "-t", "--no-title":
			argument.isNoTitle = true
		default:
			args = append([]string{arg}, args...)
			break LOOP
		}
	}

	if len(args) == 0 {
		return nil, errors.New("command is required")
	}

	argument.cmd = args[0]
	argument.args = args[1:]

	return &argument, nil
}

func main() {
	arguments, err := parseArguments(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var mode ViddyIntervalMode
	switch {
	case arguments.isPrecise:
		mode = ViddyIntervalModePrecise
	case arguments.isActual:
		mode = ViddyIntervalModeActual
	default:
		mode = ViddyIntervalModeSequential
	}

	v := NewViddy(arguments.interval, arguments.cmd, arguments.args, mode)
	v.isDebug = arguments.isDebug
	v.isNoTitle = arguments.isNoTitle
	v.isShowDiff = arguments.isDiff

	if err := v.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
