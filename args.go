package main

import (
	"errors"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

type Arguments struct {
	interval    time.Duration
	isPrecise   bool
	isClockwork bool
	isDebug     bool
	isDiff      bool
	isNoTitle   bool
	isHelp      bool
	isVersion   bool

	cmd  string
	args []string
}

var (
	NoCommand        = errors.New("command is required")
	IntervalTooSmall = errors.New("interval too small")
)

func parseArguments(args []string) (*Arguments, func(), error) {
	var argument Arguments

	flagSet := pflag.NewFlagSet("", pflag.ExitOnError)
	flagSet.DurationVarP(&argument.interval, "interval", "n", 2*time.Second, "seconds to wait between updates")
	flagSet.BoolVarP(&argument.isPrecise, "precise", "p", false, "attempt run command in precise intervals")
	flagSet.BoolVarP(&argument.isClockwork, "clockwork", "c", false, "run command in precise intervals forcibly")
	flagSet.BoolVar(&argument.isDebug, "debug", false, "")
	flagSet.BoolVarP(&argument.isDiff, "differences", "d", false, "highlight changes between updates")
	flagSet.BoolVarP(&argument.isNoTitle, "no-title", "t", false, "turn off header")
	flagSet.BoolVarP(&argument.isHelp, "help", "h", false, "display this help and exit")
	flagSet.BoolVarP(&argument.isVersion, "version", "v", false, "output version information and exit")

	if len(args) == 0 {
		return &argument, flagSet.Usage, NoCommand
	}

	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			continue
		}

		if i > 0 {
			prev := args[i-1]
			switch prev {
			case "--interval", "-n":
				continue
			}
		}

		argument.cmd = args[i]
		if i+1 <= len(args) {
			argument.args = args[i+1:]
		}

		_ = flagSet.Parse(args[:i])
		break
	}

	return &argument, flagSet.Usage, nil
}
