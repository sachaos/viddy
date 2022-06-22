package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cast"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	errNoCommand        = errors.New("command is required")
	errIntervalTooSmall = errors.New("interval too small")
)

type config struct {
	runtime runtimeConfig
	general general
	theme   theme
	keymap  keymapping
}

type runtimeConfig struct {
	cmd      string
	args     []string
	interval time.Duration
	mode     ViddyIntervalMode
	help     bool
	version  bool
}

type general struct {
	shell        string
	shellOptions string
	debug        bool
	bell         bool
	differences  bool
	noTitle      bool
	pty          bool
	unfold       bool
}

type theme struct {
	tview.Theme
}

type KeyStroke struct {
	Key     tcell.Key
	Rune    rune
	ModMask tcell.ModMask
}

type keymapping struct {
	toggleTimeMachine           map[KeyStroke]struct{}
	goToPastOnTimeMachine       map[KeyStroke]struct{}
	goToFutureOnTimeMachine     map[KeyStroke]struct{}
	goToMorePastOnTimeMachine   map[KeyStroke]struct{}
	goToMoreFutureOnTimeMachine map[KeyStroke]struct{}
	goToNowOnTimeMachine        map[KeyStroke]struct{}
	goToOldestOnTimeMachine     map[KeyStroke]struct{}
}

//nolint:funlen,cyclop
func newConfig(v *viper.Viper, args []string) (*config, error) {
	flagSet := pflag.NewFlagSet("", pflag.ExitOnError)

	// runtimeConfig
	flagSet.StringP("interval", "n", "2s", "seconds to wait between updates")
	flagSet.BoolP("precise", "p", false, "attempt run command in precise intervals")
	flagSet.BoolP("clockwork", "c", false, "run command in precise intervals forcibly")
	flagSet.BoolP("help", "h", false, "display this help and exit")
	flagSet.BoolP("version", "v", false, "output version information and exit")

	// general
	flagSet.BoolP("bell", "b", false, "ring terminal bell changes between updates")
	flagSet.BoolP("differences", "d", false, "highlight changes between updates")
	flagSet.BoolP("no-title", "t", false, "turn off header")
	flagSet.Bool("debug", false, "")
	flagSet.String("shell", "", "shell (default \"sh\")")
	flagSet.String("shell-options", "", "additional shell options")
	flagSet.Bool("unfold", false, "unfold")
	flagSet.Bool("pty", false, "run on pty (experimental)")

	flagSet.SetInterspersed(false)

	if err := flagSet.Parse(args); err != nil {
		return nil, err
	}

	var conf config

	intervalStr, _ := flagSet.GetString("interval")

	interval, err := parseInterval(intervalStr)
	if err != nil {
		return nil, err
	}

	conf.runtime.interval = interval

	conf.runtime.mode = ViddyIntervalModeSequential
	if ok, _ := flagSet.GetBool("precise"); ok {
		conf.runtime.mode = ViddyIntervalModePrecise
	}

	if ok, _ := flagSet.GetBool("clockwork"); ok {
		conf.runtime.mode = ViddyIntervalModeClockwork
	}

	conf.runtime.help, _ = flagSet.GetBool("help")
	conf.runtime.version, _ = flagSet.GetBool("version")

	if err := v.BindPFlag("general.debug", flagSet.Lookup("debug")); err != nil {
		return nil, err
	}

	if err := v.BindPFlag("general.shell", flagSet.Lookup("shell")); err != nil {
		return nil, err
	}

	v.SetDefault("general.shell", "sh")

	if err := v.BindPFlag("general.shell_options", flagSet.Lookup("shell-options")); err != nil {
		return nil, err
	}

	if err := v.BindPFlag("general.bell", flagSet.Lookup("bell")); err != nil {
		return nil, err
	}

	if err := v.BindPFlag("general.differences", flagSet.Lookup("differences")); err != nil {
		return nil, err
	}

	if err := v.BindPFlag("general.no_title", flagSet.Lookup("no-title")); err != nil {
		return nil, err
	}

	if err := v.BindPFlag("general.unfold", flagSet.Lookup("unfold")); err != nil {
		return nil, err
	}

	if err := v.BindPFlag("general.pty", flagSet.Lookup("pty")); err != nil {
		return nil, err
	}

	conf.general.debug = v.GetBool("general.debug")
	conf.general.shell = v.GetString("general.shell")
	conf.general.shellOptions = v.GetString("general.shell_options")
	conf.general.bell, _ = flagSet.GetBool("bell")
	conf.general.differences, _ = flagSet.GetBool("differences")
	conf.general.noTitle, _ = flagSet.GetBool("no-title")
	conf.general.unfold = v.GetBool("general.unfold")
	conf.general.pty = v.GetBool("general.pty")

	v.SetDefault("color.border", "gray")
	v.SetDefault("color.title", "gray")

	conf.theme.Theme = tview.Theme{
		PrimitiveBackgroundColor:    tcell.GetColor(v.GetString("color.background")),
		ContrastBackgroundColor:     tcell.GetColor(v.GetString("color.contrast_background")),
		MoreContrastBackgroundColor: tcell.GetColor(v.GetString("color.more_contrast_background")),
		BorderColor:                 tcell.GetColor(v.GetString("color.border")),
		TitleColor:                  tcell.GetColor(v.GetString("color.title")),
		GraphicsColor:               tcell.GetColor(v.GetString("color.graphics")),
		PrimaryTextColor:            tcell.GetColor(v.GetString("color.text")),
		SecondaryTextColor:          tcell.GetColor(v.GetString("color.secondary_text")),
		TertiaryTextColor:           tcell.GetColor(v.GetString("color.tertiary_text")),
		InverseTextColor:            tcell.GetColor(v.GetString("color.inverse_text")),
		ContrastSecondaryTextColor:  tcell.GetColor(v.GetString("color.contrast_secondary_text")),
	}

	conf.keymap.toggleTimeMachine = getKeymapDefault(v, "keymap.toggle_timemachine",
		map[KeyStroke]struct{}{mustParseKeymap(" "): {}})
	conf.keymap.goToPastOnTimeMachine = getKeymapDefault(v, "keymap.timemachine_go_to_past",
		map[KeyStroke]struct{}{mustParseKeymap("Shift-J"): {}})
	conf.keymap.goToFutureOnTimeMachine = getKeymapDefault(v, "keymap.timemachine_go_to_future",
		map[KeyStroke]struct{}{mustParseKeymap("Shift-K"): {}})
	conf.keymap.goToMorePastOnTimeMachine = getKeymapDefault(v, "keymap.timemachine_go_to_more_past",
		map[KeyStroke]struct{}{mustParseKeymap("Shift-F"): {}})
	conf.keymap.goToMoreFutureOnTimeMachine = getKeymapDefault(v, "keymap.timemachine_go_to_more_future",
		map[KeyStroke]struct{}{mustParseKeymap("Shift-B"): {}})
	conf.keymap.goToNowOnTimeMachine = getKeymapDefault(v, "keymap.timemachine_go_to_now",
		map[KeyStroke]struct{}{mustParseKeymap("Shift-N"): {}})
	conf.keymap.goToOldestOnTimeMachine = getKeymapDefault(v, "keymap.timemachine_go_to_oldest",
		map[KeyStroke]struct{}{mustParseKeymap("Shift-O"): {}})

	if conf.runtime.interval < 10*time.Millisecond {
		return &conf, errIntervalTooSmall
	}

	rest := flagSet.Args()

	if len(rest) == 0 {
		return &conf, errNoCommand
	}

	conf.runtime.cmd = rest[0]
	conf.runtime.args = rest[1:]

	return &conf, nil
}

func parseInterval(intervalStr string) (time.Duration, error) {
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		intervalFloat, err := strconv.ParseFloat(intervalStr, 64)
		if err != nil {
			return 0, err
		}

		interval = time.Duration(intervalFloat * float64(time.Second))
	}

	return interval, nil
}

func getKeymapDefault(v *viper.Viper, key string, d map[KeyStroke]struct{}) map[KeyStroke]struct{} {
	keymap, err := getKeymap(v, key)
	if err != nil {
		return d
	}

	return keymap
}

type cannotFindKeyError struct {
	key string
}

func (e cannotFindKeyError) Error() string {
	return fmt.Sprintf("could not find the key: %q", e.key)
}

func getKeymap(v *viper.Viper, key string) (map[KeyStroke]struct{}, error) {
	value := v.Get(key)
	if value == nil {
		return nil, cannotFindKeyError{key: key}
	}

	if k, err := cast.ToStringE(value); err == nil {
		key, err := ParseKeyStroke(k)
		if err != nil {
			return nil, err
		}

		return map[KeyStroke]struct{}{key: {}}, nil
	}

	if keys, err := cast.ToStringSliceE(value); err == nil {
		m := map[KeyStroke]struct{}{}

		for _, k := range keys {
			key, err := ParseKeyStroke(k)
			if err != nil {
				return nil, err
			}

			m[key] = struct{}{}
		}

		return m, nil
	}

	return nil, nil
}

func mustParseKeymap(key string) KeyStroke {
	keymap, err := ParseKeyStroke(key)
	if err != nil {
		panic(err)
	}

	return keymap
}

type parseKeyStrokeError struct {
	key string
}

func (e parseKeyStrokeError) Error() string {
	return fmt.Sprintf("connot parse key: %q", e.key)
}

// ParseKeyStroke parse string describing key.
func ParseKeyStroke(key string) (KeyStroke, error) {
	if len(key) == 0 {
		return KeyStroke{}, parseKeyStrokeError{key: key}
	}

	var mod tcell.ModMask

	if strings.HasPrefix(key, "Ctrl-") {
		mod |= tcell.ModCtrl
		key = strings.TrimPrefix(key, "Ctrl-")
	}

	if strings.HasPrefix(key, "Alt-") {
		mod |= tcell.ModAlt
		key = strings.TrimPrefix(key, "Alt-")
	}

	if strings.HasPrefix(key, "Shift-") {
		key = strings.TrimPrefix(key, "Shift-")

		if k, err := keyOf(key); err == nil {
			mod |= tcell.ModShift

			return KeyStroke{
				Key:     k,
				ModMask: mod,
			}, nil
		}

		k := []rune(key)[0]

		return KeyStroke{
			Key:     tcell.KeyRune,
			Rune:    unicode.ToUpper(k),
			ModMask: mod,
		}, nil
	}

	if k, err := keyOf(key); err == nil {
		return KeyStroke{
			Key:     k,
			ModMask: mod,
		}, nil
	}

	k := []rune(key)[0]

	return KeyStroke{
		Key:     tcell.KeyRune,
		Rune:    unicode.ToLower(k),
		ModMask: mod,
	}, nil
}

type keyNotFoundError struct{}

func (k keyNotFoundError) Error() string {
	return "not found"
}

func keyOf(key string) (tcell.Key, error) {
	for k, name := range tcell.KeyNames {
		if name == key {
			return k, nil
		}
	}

	return 0, keyNotFoundError{}
}

func isFlagSet(str string, flagSet *pflag.FlagSet) bool {
	res := false
	flagSet.Visit(func (f *pflag.Flag) {
		if f.Name == str {
			res = true
		}
	})

	return res
}
