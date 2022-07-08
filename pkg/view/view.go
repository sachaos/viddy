package view

import (
	"github.com/rivo/tview"
	"github.com/sachaos/viddy/pkg/config"
	"io"
	"strings"
	"time"
)

type View struct {
	app *tview.Application

	keymap   config.KeyMapping
	interval *tview.TextView
	command  *tview.TextView
	time     *tview.TextView
	body     *tview.TextView
	help     *tview.TextView
	history  *HistoryView

	showHelpView bool
	noTitle bool
	timeMachine bool
	editQuery bool
	query string
}

func NewView(i time.Duration, cmd string, args []string, helpPage string, historyView *HistoryView) *View {
	app := tview.NewApplication()
	app.EnableMouse(true)

	interval := tview.NewTextView()
	interval.SetBorder(true).SetTitle("Every")
	interval.SetTitleAlign(tview.AlignLeft)
	interval.SetText(i.String())

	c := []string{cmd}
	c = append(c, args...)
	command := tview.NewTextView()
	command.SetBorder(true).SetTitle("Command")
	command.SetTitleAlign(tview.AlignLeft)
	command.SetText(strings.Join(c, " "))

	t := tview.NewTextView()
	t.SetBorder(true).SetTitle("Time")
	t.SetTitleAlign(tview.AlignLeft)

	b := tview.NewTextView()
	b.SetDynamicColors(true)
	b.SetRegions(true)
	b.GetInnerRect()

	h := tview.NewTextView()
	h.SetDynamicColors(true)
	_, _ = io.WriteString(h, helpPage)

	return &View{
		app: app,
		interval: interval,
		command: command,
		time: t,
		body: b,
		history: historyView,

		help: h,
	}
}

func (v *View) Run() error {
	return v.app.Run()
}

func (v *View) SetWrap(b bool) {
	v.body.SetWrap(b)
	v.arrange()
}

func (v *View) SetShowHelp(b bool) {
	v.showHelpView = b
	v.arrange()
}

func (v *View) SetNoTitle(b bool) {
	v.noTitle = b
	v.arrange()
}

func (v *View) SetTimeMachine(b bool)  {
	v.timeMachine = b
	v.arrange()
}

func (v *View) SetEditQuery(b bool)  {
	v.arrange()
}

func (v *View) arrange() {
	if v.showHelpView {
		v.app.SetRoot(v.help, true)
		return
	}

	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	if !v.noTitle {
		title := tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(v.interval, 10, 1, false).
			AddItem(v.command, 0, 1, false).
			AddItem(v.time, 21, 1, false)

		flex.AddItem(title, 3, 1, false)
	}

	body := tview.NewFlex().SetDirection(tview.FlexRow)
	body.AddItem(v.body, 0, 1, false)

	middle := tview.NewFlex().SetDirection(tview.FlexColumn)
	middle.AddItem(body, 0, 1, false)

	if v.timeMachine {
		middle.AddItem(v.history, 21, 1, true)
	}

	flex.AddItem(
		middle,
		0, 1, false)

	bottom := tview.NewFlex().SetDirection(tview.FlexColumn)

	if v.editQuery || v.query != "" {
		bottom.AddItem(v.queryEditor, 0, 1, false)
	} else {
		bottom.AddItem(tview.NewBox(), 0, 1, false)
	}
}
