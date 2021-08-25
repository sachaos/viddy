package main

import (
	"fmt"
	"os"
)

var version string

func main() {
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

	v := NewViddy(arguments.interval, arguments.cmd, arguments.args, arguments.shell, arguments.shellOpts, mode)
	v.isDebug = arguments.isDebug
	v.isNoTitle = arguments.isNoTitle
	v.isShowDiff = arguments.isDiff

	if err := v.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
