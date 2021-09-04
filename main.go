package main

import (
	"fmt"
	"os"

	"github.com/rivo/tview"

	"github.com/adrg/xdg"
	"github.com/spf13/viper"
)

var version string

var DefaultTheme = tview.Theme{}

func main() {
	v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigName("viddy")
	v.AddConfigPath(xdg.ConfigHome)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			v.SafeWriteConfig()
		} else {
			fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
			os.Exit(1)
		}
	}

	arguments, err := parseArguments(os.Args[1:])
	if arguments.isHelp {
		help()
		os.Exit(0)
	}

	if arguments.isVersion {
		fmt.Printf("viddy version: %s\n", version)
		os.Exit(0)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var mode ViddyIntervalMode

	switch {
	case arguments.isPrecise:
		mode = ViddyIntervalModePrecise
	case arguments.isClockwork:
		mode = ViddyIntervalModeClockwork
	default:
		mode = ViddyIntervalModeSequential
	}

	tview.Styles = DefaultTheme

	app := NewViddy(arguments.interval, arguments.cmd, arguments.args, arguments.shell, arguments.shellOpts, mode)
	app.isDebug = arguments.isDebug
	app.isNoTitle = arguments.isNoTitle
	app.isShowDiff = arguments.isDiff

	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
