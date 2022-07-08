package view

import (
	"bytes"
	"github.com/gdamore/tcell/v2"
	"github.com/sachaos/viddy/pkg/config"
	"strings"
	"text/template"
)

func HelpPage(keymap config.KeyMapping) string {
	helpTemplate := `Press ESC or Q to go back

 [::b]Key Bindings[-:-:-]

   [::u]General[-:-:-]     

   Toggle time machine mode  : [yellow]SPACE[-:-:-]
   Toggle suspend execution  : [yellow]s[-:-:-]
   Toggle ring terminal bell : [yellow]b[-:-:-]
   Toggle diff               : [yellow]d[-:-:-]
   Toggle header display     : [yellow]t[-:-:-]
   Toggle help view          : [yellow]?[-:-:-]

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

	value := struct {
		GoToPast       string
		GoToFuture     string
		GoToMorePast   string
		GoToMoreFuture string
		GoToOldest     string
		GoToNow        string
	}{
		GoToPast:       keysToString(keymap.GoToPastOnTimeMachine),
		GoToFuture:     keysToString(keymap.GoToFutureOnTimeMachine),
		GoToMorePast:   keysToString(keymap.GoToMorePastOnTimeMachine),
		GoToMoreFuture: keysToString(keymap.GoToMoreFutureOnTimeMachine),
		GoToOldest:     keysToString(keymap.GoToOldestOnTimeMachine),
		GoToNow:        keysToString(keymap.GoToNowOnTimeMachine),
	}

	var b bytes.Buffer

	tpl, _ := template.New("").Parse(helpTemplate)
	_ = tpl.Execute(&b, value)

	return b.String()
}

func keysToString(keys map[config.KeyStroke]struct{}) string {
	str := make([]string, 0, len(keys))
	for stroke := range keys {
		str = append(str, formatKeyStroke(stroke))
	}

	return strings.Join(str, ", ")
}

func formatKeyStroke(stroke config.KeyStroke) string {
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