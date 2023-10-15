package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/moby/term"
	"github.com/rivo/tview"
)

type HistoryRow struct {
	id *tview.TableCell

	addition *tview.TableCell
	deletion *tview.TableCell
	exitCode *tview.TableCell
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

	// bWidth store current pty width.
	bWidth atomic.Value

	// id -> row count (as of just after the snapshot was added).
	historyRowCount map[int64]int

	bodyView    *tview.TextView
	app         *tview.Application
	logView     *tview.TextView
	helpView    *tview.TextView
	statusView  *tview.TextView
	queryEditor *tview.InputField

	snapshotQueue    <-chan *Snapshot
	isSuspendedQueue chan<- bool
	queue            chan int64
	finishedQueue    chan int64
	diffQueue        chan int64

	currentID        int64
	latestFinishedID int64
	isTimeMachine    bool
	isSuspend        bool
	isNoTitle        bool
	isRingBell       bool
	isShowDiff       bool
	skipEmptyDiffs   bool
	isEditQuery      bool
	unfold           bool
	pty              bool

	query string

	isDebug      bool
	showLogView  bool
	showHelpView bool
}

type ViddyIntervalMode string

var (
	// ViddyIntervalModeClockwork is mode to run command in precise intervals forcibly.
	ViddyIntervalModeClockwork ViddyIntervalMode = "clockwork"

	// ViddyIntervalModePrecise is mode to run command in precise intervals like `watch -p`.
	ViddyIntervalModePrecise ViddyIntervalMode = "precise"

	// ViddyIntervalModeSequential is mode to run command sequential like `watch` command.
	ViddyIntervalModeSequential ViddyIntervalMode = "sequential"

	errCannotCreateSnapshot = errors.New("cannot find the snapshot")
	errNotCompletedYet      = errors.New("not completed yet")
)

func NewViddy(conf *config) *Viddy {
	begin := time.Now().UnixNano()

	newSnap := func(id int64, before *Snapshot, finish chan<- struct{}) *Snapshot {
		return NewSnapshot(id, conf.runtime.cmd, conf.runtime.args, conf.general.shell, conf.general.shellOptions, before, finish)
	}

	var (
		snapshotQueue    <-chan *Snapshot
		isSuspendedQueue chan<- bool
	)

	switch conf.runtime.mode {
	case ViddyIntervalModeClockwork:
		snapshotQueue, isSuspendedQueue = ClockSnapshot(begin, newSnap, conf.runtime.interval)
	case ViddyIntervalModeSequential:
		snapshotQueue, isSuspendedQueue = SequentialSnapshot(newSnap, conf.runtime.interval)
	case ViddyIntervalModePrecise:
		snapshotQueue, isSuspendedQueue = PreciseSnapshot(newSnap, conf.runtime.interval)
	}

	return &Viddy{
		keymap: conf.keymap,

		begin:       begin,
		cmd:         conf.runtime.cmd,
		args:        conf.runtime.args,
		duration:    conf.runtime.interval,
		snapshots:   sync.Map{},
		historyRows: map[int64]*HistoryRow{},

		historyRowCount: map[int64]int{},

		snapshotQueue:    snapshotQueue,
		isSuspendedQueue: isSuspendedQueue,
		queue:            make(chan int64),
		finishedQueue:    make(chan int64),
		diffQueue:        make(chan int64, 100),

		isRingBell:     conf.general.bell,
		isShowDiff:     conf.general.differences,
		skipEmptyDiffs: conf.general.skipEmptyDiffs,
		isNoTitle:      conf.general.noTitle,
		isDebug:        conf.general.debug,
		unfold:         conf.general.unfold,
		pty:            conf.general.pty,

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
	v.setSelection(v.currentID, -1)
	v.arrange()
}

func (v *Viddy) SetIsTimeMachine(b bool) {
	v.isTimeMachine = b
	if !v.isTimeMachine {
		v.setSelection(v.latestFinishedID, -1)
	} else {
		v.goToNowOnTimeMachine()
	}

	v.arrange()
}

func (v *Viddy) println(a ...interface{}) {
	_, _ = fmt.Fprintln(v.logView, a...)
}

func (v *Viddy) addSnapshot(s *Snapshot) {
	v.snapshots.Store(s.id, s)
}

func (v *Viddy) startRunner() {
	for s := range v.snapshotQueue {
		v.addSnapshot(s)
		v.queue <- s.id

		_ = s.run(v.finishedQueue, v.getBodyWidth(), v.pty)
	}
}

func (v *Viddy) updateSelection() {
	if !v.isTimeMachine {
		v.setSelection(v.latestFinishedID, -1)
	} else {
		v.setSelection(v.currentID, -1)
	}
}

func (v *Viddy) addSnapshotToView(id int64, r *HistoryRow) {
	v.historyView.InsertRow(0)
	v.historyView.SetCell(0, 0, r.id)
	v.historyView.SetCell(0, 1, r.addition)
	v.historyView.SetCell(0, 2, r.deletion)
	v.historyView.SetCell(0, 3, r.exitCode)

	v.historyRowCount[id] = v.historyView.GetRowCount()

	v.updateSelection()
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

			if s.diffAdditionCount > 0 || s.diffDeletionCount > 0 {
				if v.isRingBell {
					fmt.Print(string(byte(7)))
				}
			} else if v.skipEmptyDiffs {
				return
			}

			r, ok := v.historyRows[id]
			if !ok {
				return
			}

			// if skipEmptyDiffs is true, queueHandler wouldn't have added the
			// snapshot to view, so we need to add it here.
			if v.skipEmptyDiffs {
				v.addSnapshotToView(id, r)
			}

			r.addition.SetText("+" + strconv.Itoa(s.diffAdditionCount))
			r.deletion.SetText("-" + strconv.Itoa(s.diffDeletionCount))
		}()
	}
}

//nolint:funlen
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

				if s.exitCode > 0 {
					r.exitCode.SetText(fmt.Sprintf("E(%d)", s.exitCode))
				}

				ls := v.getSnapShot(v.latestFinishedID)
				if ls == nil || s.start.After(ls.start) {
					v.latestFinishedID = id
					v.updateSelection()
				}
			case id := <-v.queue:
				s := v.getSnapShot(id)
				idCell := tview.NewTableCell(strconv.FormatInt(s.id, 10)).SetTextColor(tview.Styles.SecondaryTextColor)
				additionCell := tview.NewTableCell("").SetTextColor(tcell.ColorGreen)
				deletionCell := tview.NewTableCell("").SetTextColor(tcell.ColorRed)
				exitCodeCell := tview.NewTableCell("").SetTextColor(tcell.ColorYellow)

				r := &HistoryRow{
					id:       idCell,
					addition: additionCell,
					deletion: deletionCell,
					exitCode: exitCodeCell,
				}
				v.historyRows[s.id] = r

				// if skipEmptyDiffs is true, we need to check if the snapshot
				// is empty before adding it to the view (in diffQueueHandler).
				//
				// This means we're trading off two things:
				//
				// 1. We're not showing the snapshot in history view until the
				//    command finishes running, which means it's not possible
				//    to see partial output.
				// 2. Order of the snapshots in history view is lost
				//    (in non-sequential modes), as some commands could finish
				//    running quicker than others for whatever reason.
				//
				// It of course is possible to address these issues by adding
				// all snapshots to the history view and then removing the empty
				// ones but it unnecessarily complicates the implementation.
				if !v.skipEmptyDiffs {
					v.addSnapshotToView(id, r)
				}
			}
		}()
	}
}

// setSelection selects the given row in the history view. If row is -1, it will
// attempt to select the row corresponding to the given id (or default to the
// latest row if id doesn't exist).
func (v *Viddy) setSelection(id int64, row int) {
	if id == -1 {
		return
	}

	v.historyView.ScrollToBeginning()

	isSelectable, _ := v.historyView.GetSelectable()
	if !isSelectable {
		v.historyView.SetSelectable(true, false)
	}

	if row == -1 {
		row = v.historyView.GetRowCount() - v.historyRowCount[id]
	}

	v.historyView.Select(row, 0)
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

	bw := v.bodyView.BatchWriter()
	defer bw.Close()

	bw.Clear()

	if !s.completed {
		return errNotCompletedYet
	}

	return s.render(bw, v.isShowDiff, v.query)
}

func (v *Viddy) UpdateStatusView() {
	v.statusView.SetText(fmt.Sprintf("Suspend %s  Diff %s  Bell %s",
		convertToOnOrOff(v.isSuspend),
		convertToOnOrOff(v.isShowDiff),
		convertToOnOrOff(v.isRingBell)))
}

func convertToOnOrOff(on bool) string {
	if on {
		return "[green]◯[reset]"
	}

	return "[red]◯[reset]"
}

func (v *Viddy) arrange() {
	if v.showHelpView {
		v.app.SetRoot(v.helpView, true)

		return
	}

	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	if !v.isNoTitle {
		title := tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(v.intervalView, 10, 1, false).
			AddItem(v.commandView, 0, 1, false).
			AddItem(v.timeView, 21, 1, false)

		flex.AddItem(title, 3, 1, false)
	}

	body := tview.NewFlex().SetDirection(tview.FlexRow)
	body.AddItem(v.bodyView, 0, 1, false)

	middle := tview.NewFlex().SetDirection(tview.FlexColumn)
	middle.AddItem(body, 0, 1, false)

	if v.isTimeMachine {
		middle.AddItem(v.historyView, 21, 1, true)
	}

	flex.AddItem(
		middle,
		0, 1, false)

	bottom := tview.NewFlex().SetDirection(tview.FlexColumn)

	if v.isEditQuery || v.query != "" {
		bottom.AddItem(v.queryEditor, 0, 1, false)
	} else {
		bottom.AddItem(tview.NewBox(), 0, 1, false)
	}

	bottom.AddItem(v.statusView, 25, 1, false)

	flex.AddItem(bottom, 1, 1, false)

	if v.showLogView {
		flex.AddItem(v.logView, 10, 1, false)
	}

	v.app.SetRoot(flex, true)
}

// Run is entry point to run viddy.
//
//nolint:funlen,gocognit,cyclop,gocyclo,maintidx
func (v *Viddy) Run() error {
	b := tview.NewTextView()
	b.SetDynamicColors(true)
	b.SetRegions(true)
	b.GetInnerRect()
	b.SetWrap(!v.unfold)
	v.bodyView = b

	t := tview.NewTextView()
	t.SetBorder(true).SetTitle("Time")
	t.SetTitleAlign(tview.AlignLeft)
	v.timeView = t

	h := tview.NewTable()
	v.historyView = h
	h.SetTitle("History")
	h.SetTitleAlign(tview.AlignLeft)
	h.SetBorder(true)
	h.ScrollToBeginning()
	h.SetSelectionChangedFunc(func(row, column int) {
		c := v.historyView.GetCell(row, column)
		id, err := strconv.ParseInt(c.Text, 10, 64)
		if err == nil {
			_ = v.renderSnapshot(id)
		}
	})
	h.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorGray))

	var cmd []string
	cmd = append(cmd, v.cmd)
	cmd = append(cmd, v.args...)

	c := tview.NewTextView()
	c.SetBorder(true).SetTitle("Command")
	c.SetTitleAlign(tview.AlignLeft)
	c.SetText(strings.Join(cmd, " "))
	v.commandView = c

	d := tview.NewTextView()
	d.SetBorder(true).SetTitle("Every")
	d.SetTitleAlign(tview.AlignLeft)
	d.SetText(v.duration.String())
	v.intervalView = d

	s := tview.NewTextView()
	s.SetDynamicColors(true)
	v.statusView = s

	l := tview.NewTextView()
	l.SetBorder(true).SetTitle("Log")
	l.ScrollToEnd()
	v.logView = l

	hv := tview.NewTextView()
	hv.SetDynamicColors(true)
	_, _ = io.WriteString(hv, v.helpPage())
	v.helpView = hv

	q := tview.NewInputField().SetLabel("/")
	q.SetChangedFunc(func(text string) {
		v.query = text
		_ = v.renderSnapshot(v.currentID)
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

		if event.Key() == tcell.KeyEsc {
			v.showHelpView = false
			v.arrange()
		}

		if event.Rune() == 'q' {
			if v.showHelpView { // if it's help mode, just go back
				v.showHelpView = false
				v.arrange()
			} else { // it's not help view, so just quit
				v.app.Stop()
				os.Exit(0)
			}
		}

		// quit viddy from any view
		if event.Rune() == 'Q' {
			v.app.Stop()
			os.Exit(0)
		}

		switch event.Rune() {
		case 's':
			v.isSuspend = !v.isSuspend
			v.isSuspendedQueue <- v.isSuspend
		case 'b':
			v.isRingBell = !v.isRingBell
		case 'd':
			v.SetIsShowDiff(!v.isShowDiff)
		case 't':
			v.SetIsNoTitle(!v.isNoTitle)
		case 'u':
			b.SetWrap(v.unfold)
			v.unfold = !v.unfold
		case 'x':
			if v.isDebug {
				v.ShowLogView(!v.showLogView)
			}
		case '?':
			v.ShowHelpView(!v.showHelpView)
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

	app.SetAfterDrawFunc(func(screen tcell.Screen) {
		v.setBodyWidth()
	})

	v.UpdateStatusView()

	v.app = app
	v.arrange()

	v.setBodyWidth()

	go v.diffQueueHandler()
	go v.queueHandler()
	go v.startRunner()

	return app.Run()
}

func (v *Viddy) goToRow(row int) {
	if row < 0 {
		row = 0
	} else if count := v.historyView.GetRowCount(); row >= count {
		row = count - 1
	}

	var (
		cell    = v.historyView.GetCell(row, 0)
		id, err = strconv.ParseInt(cell.Text, 10, 64)
	)

	if err == nil { // if _no_ error
		v.setSelection(id, row)
	}
}

func (v *Viddy) goToPastOnTimeMachine() {
	selection, _ := v.historyView.GetSelection()
	v.goToRow(selection + 1)
}

func (v *Viddy) goToFutureOnTimeMachine() {
	selection, _ := v.historyView.GetSelection()
	v.goToRow(selection - 1)
}

func (v *Viddy) goToMorePastOnTimeMachine() {
	selection, _ := v.historyView.GetSelection()
	v.goToRow(selection + 10)
}

func (v *Viddy) goToMoreFutureOnTimeMachine() {
	selection, _ := v.historyView.GetSelection()
	v.goToRow(selection - 10)
}

func (v *Viddy) goToNowOnTimeMachine() {
	v.goToRow(0)
}

func (v *Viddy) goToOldestOnTimeMachine() {
	v.goToRow(v.historyView.GetRowCount() - 1)
}

var helpTemplate = `Press ESC or q to go back

 [::b]Key Bindings[-:-:-]

   [::u]General[-:-:-]

   Toggle time machine mode  : [yellow]SPACE[-:-:-]
   Toggle suspend execution  : [yellow]s[-:-:-]
   Toggle ring terminal bell : [yellow]b[-:-:-]
   Toggle diff               : [yellow]d[-:-:-]
   Toggle header display     : [yellow]t[-:-:-]
   Toggle help view          : [yellow]?[-:-:-]
   Toggle unfold             : [yellow]u[-:-:-]
   Quit Viddy                : [yellow]Q[-:-:-]

   [::u]Pager[-:-:-]

   Search text              : [yellow]/[-:-:-]
   Move to next line        : [yellow]j[-:-:-]
   Move to previous line    : [yellow]k[-:-:-]
   Page down                : [yellow]Ctrl-F[-:-:-]
   Page up                  : [yellow]Ctrl-B[-:-:-]
   Go to top of page        : [yellow]g[-:-:-]
   Go to bottom of page     : [yellow]G[-:-:-]

   [::u]Time machine[-:-:-]

   Go to the past            : [yellow]{{ .GoToPast }}[-:-:-]
   Back to the future        : [yellow]{{ .GoToFuture }}[-:-:-]
   Go to more past           : [yellow]{{ .GoToMorePast }}[-:-:-]
   Back to more future       : [yellow]{{ .GoToMoreFuture }}[-:-:-]
   Go to oldest position     : [yellow]{{ .GoToOldest }}[-:-:-]
   Back to current position  : [yellow]{{ .GoToNow }}[-:-:-]
`

func keysToString(keys map[KeyStroke]struct{}) string {
	str := make([]string, 0, len(keys))
	for stroke := range keys {
		str = append(str, formatKeyStroke(stroke))
	}

	return strings.Join(str, ", ")
}

func formatKeyStroke(stroke KeyStroke) string {
	var b strings.Builder
	if stroke.ModMask&tcell.ModCtrl != 0 {
		b.WriteString("Ctrl-")
	}

	if stroke.ModMask&tcell.ModAlt != 0 {
		b.WriteString("Alt-")
	}

	if stroke.ModMask&tcell.ModShift != 0 {
		b.WriteString("Shift-")
	}

	if stroke.Key == tcell.KeyRune {
		b.WriteString(string(stroke.Rune))
	} else {
		b.WriteString(tcell.KeyNames[stroke.Key])
	}

	return b.String()
}

func (v *Viddy) setBodyWidth() {
	width := 80
	if winsize, err := term.GetWinsize(os.Stdout.Fd()); err == nil {
		width = int(winsize.Width)
	}

	v.bWidth.Store(width)
}

func (v *Viddy) getBodyWidth() int {
	return v.bWidth.Load().(int)
}

func (v *Viddy) helpPage() string {
	value := struct {
		GoToPast       string
		GoToFuture     string
		GoToMorePast   string
		GoToMoreFuture string
		GoToOldest     string
		GoToNow        string
	}{
		GoToPast:       keysToString(v.keymap.goToPastOnTimeMachine),
		GoToFuture:     keysToString(v.keymap.goToFutureOnTimeMachine),
		GoToMorePast:   keysToString(v.keymap.goToMorePastOnTimeMachine),
		GoToMoreFuture: keysToString(v.keymap.goToMoreFutureOnTimeMachine),
		GoToOldest:     keysToString(v.keymap.goToOldestOnTimeMachine),
		GoToNow:        keysToString(v.keymap.goToNowOnTimeMachine),
	}

	var b bytes.Buffer

	tpl, _ := template.New("").Parse(helpTemplate)
	_ = tpl.Execute(&b, value)

	return b.String()
}

func (v *Viddy) ShowHelpView(b bool) {
	v.showHelpView = b
	v.arrange()
}
