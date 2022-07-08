package view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type QueryView struct {

}

func NewQueryView() *QueryView {
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
}
