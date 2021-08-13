package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Snapshot struct {
	id int64

	command string
	args    []string

	result bytes.Buffer
	start  time.Time
	end    time.Time

	completed bool
	err       error

	before *Snapshot
	finish chan<- struct{}
}

func NewSnapshot(id int64, command string, args []string, before *Snapshot, finish chan<- struct{}) *Snapshot {
	return &Snapshot{
		id:      id,
		command: command,
		args:    args,

		before: before,
		finish: finish,
	}
}

func (s *Snapshot) run(finishedQueue chan <- int64) error {
	s.start = time.Now()
	defer func() {
		s.end = time.Now()
	}()

	command := exec.Command(s.command, s.args...)
	command.Stdout = &s.result

	if err := command.Start(); err != nil {
		return nil
	}

	go func() {
		if err := command.Wait(); err != nil {
			s.err = err
		}

		s.completed = true
		finishedQueue <- s.id
		close(s.finish)
	}()

	return nil
}

func (s *Snapshot) render(w io.Writer) error {
	_, err := io.Copy(tview.ANSIWriter(w), &s.result)
	return err
}

type Viddy struct {
	cmd  string
	args []string

	duration  time.Duration
	snapshots map[int64]*Snapshot

	timeView     *tview.TextView
	historyView  *tview.Table
	historyCells map[int64]*tview.TableCell
	sync.RWMutex

	idList []int64

	bodyView    *tview.TextView
	app         *tview.Application
	logView     *tview.TextView
	statusView  *tview.TextView

	snapshotQueue <-chan *Snapshot
	queue         chan int64
	finishedQueue chan int64

	latestFinishedID int64
	isTimemachine    bool
}

func ClockSnapshot(name string, args []string, interval time.Duration) <- chan *Snapshot {
	c := make(chan *Snapshot)

	go func() {
		var s *Snapshot
		t := time.Tick(interval)

		for {
			select {
			case now := <- t:
				finish := make(chan struct{})
				id := now.UnixNano()
				s = NewSnapshot(id, name, args, s, finish)
				c <- s
			}
		}
	}()

	return c
}

func PreciseSnapshot(name string, args []string, interval time.Duration) <- chan *Snapshot {
	c := make(chan *Snapshot)

	go func() {
		var s *Snapshot

		for {
			finish := make(chan struct{})
			start := time.Now()
			id := start.UnixNano()
			s = NewSnapshot(id, name, args, s, finish)
			c <- s
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

func SequentialSnapshot(name string, args []string, interval time.Duration) <- chan *Snapshot {
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
	ViddyIntervalModeActual ViddyIntervalMode = "actual"
	ViddyIntervalModePrecise ViddyIntervalMode = "precise"
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
		duration: duration,
		snapshots: map[int64]*Snapshot{},
		historyCells: map[int64]*tview.TableCell{},

		snapshotQueue: snapshotQueue,
		queue: make(chan int64),
		finishedQueue: make(chan int64),
	}
}

func (v *Viddy) println(a ...interface{}) {
	fmt.Fprintln(v.logView, a...)
	v.app.Draw()
}

func (v *Viddy) addSnapshot(s *Snapshot) {
	v.snapshots[s.id] = s
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
		func () {
			defer v.app.Draw()

			select {
			case id := <- v.finishedQueue:
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
					v.setSelection(id)
				}
			case id := <- v.queue:
				s := v.getSnapShot(id)
				c := tview.NewTableCell(strconv.FormatInt(s.id, 10))
				v.historyCells[s.id] = c

				c.SetTextColor(tcell.ColorDarkGray)

				v.historyView.InsertRow(0)
				v.historyView.SetCell(0, 0, c)

				v.Lock()
				v.idList = append(v.idList, id)
				v.Unlock()

				v.setSelection(v.latestFinishedID)
			}
		}()
	}
}

func (v *Viddy) setSelection(id int64)  {
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
	v.timeView.SetText(strconv.FormatInt(id, 10))
}

func (v *Viddy) getSnapShot(id int64) *Snapshot {
	s, ok := v.snapshots[id]
	if !ok {
		return nil
	}

	return s
}

func (v *Viddy) renderSnapshot(id int64) error {
	s := v.getSnapShot(id)
	if s == nil {
		return errors.New("cannot find the snapshot")
	}

	if !s.completed {
		return errors.New("not completed yet")
	}

	v.println("render id:", id)
	v.bodyView.Clear()
	return s.render(v.bodyView)
}

func (v *Viddy) UpdateStatusView() {
	v.statusView.SetText(fmt.Sprintf("Timemachine: %s", convertToOnOrOff(v.isTimemachine)))
}

func convertToOnOrOff(on bool) string {
	if on {
		return "[green]On [white]"
	} else {
		return "[red]Off [white]"
	}
}

func (v *Viddy) Run() error {
	b := tview.NewTextView()
	b.SetDynamicColors(true)
	b.SetTitle("body").SetBorder(true)

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
	v.bodyView = b

	var cmd []string
	cmd = append(cmd, v.cmd)
	cmd = append(cmd, v.args...)

	c := tview.NewTextView()
	c.SetBorder(true).SetTitle("Command")
	c.SetText(strings.Join(cmd, " "))

	d := tview.NewTextView()
	d.SetBorder(true).SetTitle("Every")
	d.SetText(v.duration.String())

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
			v.isTimemachine = !v.isTimemachine
		}

		v.UpdateStatusView()

		return event
	})
	v.app = app

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(d, 10, 1, false).
				AddItem(c, 0, 1, false).
				AddItem(s, 0, 1, false).
				AddItem(t, 20, 1, false),
			3, 1, false).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(b, 0, 1, false).
				AddItem(h, 20, 1, true),
			0, 1, false).
		AddItem(l, 10, 1, false)

	go v.queueHandler()
	go v.startRunner()

	if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
		return err
	}

	return nil
}

type Arguments struct {
	interval  time.Duration
	isPrecise bool
	isActual  bool

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
	if err := v.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
