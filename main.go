package main

import (
	"fmt"
	"os"
)

var version string

func main() {
	arguments, help, err := parseArguments(os.Args[1:])
	if err == NoCommand {
		if arguments.isHelp {
			help()
			os.Exit(0)
		}

		if arguments.isVersion {
			fmt.Printf("viddy version: %s\n", version)
			os.Exit(0)
		}
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

	v := NewViddy(arguments.interval, arguments.cmd, arguments.args, mode)
	v.isDebug = arguments.isDebug
	v.isNoTitle = arguments.isNoTitle
	v.isShowDiff = arguments.isDiff

	if err := v.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
