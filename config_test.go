package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/stretchr/testify/assert"

	"github.com/spf13/viper"

	"github.com/rivo/tview"
)

func Test_newConfig(t *testing.T) {
	defaultConfig := config{
		runtime: runtimeConfig{
			cmd:         "",
			args:        nil,
			interval:    2 * time.Second,
			mode:        ViddyIntervalModeSequential,
			differences: false,
			noTitle:     false,
			help:        false,
			version:     false,
		},
		general: general{
			shell:        "sh",
			shellOptions: "",
			debug:        false,
		},
		theme: theme{
			Theme: tview.Theme{
				PrimitiveBackgroundColor:    0,
				ContrastBackgroundColor:     0,
				MoreContrastBackgroundColor: 0,
				BorderColor:                 0,
				TitleColor:                  0,
				GraphicsColor:               0,
				PrimaryTextColor:            0,
				SecondaryTextColor:          0,
				TertiaryTextColor:           0,
				InverseTextColor:            0,
				ContrastSecondaryTextColor:  0,
			},
		},
		keymap: keymapping{
			toggleTimeMachine:           map[KeyStroke]struct{}{mustParseKeymap(" "): {}},
			goToPastOnTimeMachine:       map[KeyStroke]struct{}{mustParseKeymap("Shift-J"): {}},
			goToFutureOnTimeMachine:     map[KeyStroke]struct{}{mustParseKeymap("Shift-K"): {}},
			goToMorePastOnTimeMachine:   map[KeyStroke]struct{}{mustParseKeymap("Shift-F"): {}},
			goToMoreFutureOnTimeMachine: map[KeyStroke]struct{}{mustParseKeymap("Shift-B"): {}},
		},
	}

	tests := []struct {
		name       string
		configFile string
		args       []string
		want       config
		expErr     error
	}{
		{
			name:       "default",
			configFile: "",
			args:       []string{"ls"},
			want: func() config {
				c := defaultConfig
				c.runtime.cmd = "ls"
				c.runtime.args = []string{}
				return c
			}(),
		},
		{
			name:       "help",
			configFile: "",
			args:       []string{"-h"},
			want: func() config {
				c := defaultConfig
				c.runtime.help = true
				return c
			}(),
			expErr: errNoCommand,
		},
		{
			name:       "version",
			configFile: "",
			args:       []string{"-v"},
			want: func() config {
				c := defaultConfig
				c.runtime.version = true
				return c
			}(),
			expErr: errNoCommand,
		},
		{
			name:       "interval in watch mode",
			configFile: "",
			args:       []string{"-n", "0.5", "ls"},
			want: func() config {
				c := defaultConfig
				c.runtime.cmd = "ls"
				c.runtime.args = []string{}
				c.runtime.interval = 500 * time.Millisecond
				return c
			}(),
			expErr: nil,
		},
		{
			name:       "interval in go mode",
			configFile: "",
			args:       []string{"-n", "500ms", "ls"},
			want: func() config {
				c := defaultConfig
				c.runtime.cmd = "ls"
				c.runtime.args = []string{}
				c.runtime.interval = 500 * time.Millisecond
				return c
			}(),
			expErr: nil,
		},
		{
			name: "set shell on config",
			configFile: `
[general]
shell = "zsh"
`,
			args: []string{"ls"},
			want: func() config {
				c := defaultConfig
				c.runtime.cmd = "ls"
				c.runtime.args = []string{}
				c.general.shell = "zsh"
				return c
			}(),
			expErr: nil,
		},
		{
			name: "key mapping",
			configFile: `
[keymap]
toggle_timemachine = "a"
timemachine_go_to_past = "Down"
timemachine_go_to_future = "Up"
timemachine_go_to_more_past = "Shift-Down"
timemachine_go_to_more_future = "Shift-Up"
`,
			args: []string{"ls"},
			want: func() config {
				c := defaultConfig
				c.runtime.cmd = "ls"
				c.runtime.args = []string{}

				c.keymap.toggleTimeMachine = map[KeyStroke]struct{}{KeyStroke{
					Key:  tcell.KeyRune,
					Rune: 'a',
				}: {}}
				c.keymap.goToPastOnTimeMachine = map[KeyStroke]struct{}{KeyStroke{
					Key: tcell.KeyDown,
				}: {}}
				c.keymap.goToFutureOnTimeMachine = map[KeyStroke]struct{}{KeyStroke{
					Key: tcell.KeyUp,
				}: {}}
				c.keymap.goToMorePastOnTimeMachine = map[KeyStroke]struct{}{KeyStroke{
					Key:     tcell.KeyDown,
					ModMask: tcell.ModShift,
				}: {}}
				c.keymap.goToMoreFutureOnTimeMachine = map[KeyStroke]struct{}{KeyStroke{
					Key:     tcell.KeyUp,
					ModMask: tcell.ModShift,
				}: {}}
				return c
			}(),
			expErr: nil,
		},
		{
			name: "color",
			configFile: `
[color]
background = "black"
text = "white"
`,
			args: []string{"ls"},
			want: func() config {
				c := defaultConfig
				c.runtime.cmd = "ls"
				c.runtime.args = []string{}

				c.theme.PrimitiveBackgroundColor = tcell.ColorBlack
				c.theme.PrimaryTextColor = tcell.ColorWhite
				return c
			}(),
			expErr: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			v.SetConfigType("toml")
			assert.NoError(t, v.ReadConfig(bytes.NewBufferString(tt.configFile)))

			got, err := newConfig(v, tt.args)
			assert.Equal(t, tt.expErr, err)
			assert.Equal(t, &tt.want, got)
		})
	}
}

func TestParseKeyStroke(t *testing.T) {
	tests := []struct {
		key     string
		want    KeyStroke
		wantErr bool
	}{
		{
			key: "Shift-j",
			want: KeyStroke{
				Key:     tcell.KeyRune,
				Rune:    'J',
				ModMask: 0,
			},
		},
		{
			key: "j",
			want: KeyStroke{
				Key:     tcell.KeyRune,
				Rune:    'j',
				ModMask: 0,
			},
		},
		{
			key: "Up",
			want: KeyStroke{
				Key:     tcell.KeyUp,
				Rune:    0,
				ModMask: 0,
			},
		},
		{
			key: "Shift-Up",
			want: KeyStroke{
				Key:     tcell.KeyUp,
				Rune:    0,
				ModMask: tcell.ModShift,
			},
		},
		{
			key: "Ctrl-Shift-Up",
			want: KeyStroke{
				Key:     tcell.KeyUp,
				Rune:    0,
				ModMask: tcell.ModShift | tcell.ModCtrl,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.key, func(t *testing.T) {
			got, err := ParseKeyStroke(tt.key)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
