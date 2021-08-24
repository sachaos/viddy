package main

import (
	"errors"
	"fmt"
	"strconv"
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

	shell     string
	shellOpts string

	cmd  string
	args []string
}

var (
	errNoCommand        = errors.New("command is required")
	errIntervalTooSmall = errors.New("interval too small")
)

func parseArguments(args []string) (*Arguments, error) {
	var argument Arguments

	var intervalStr string

	flagSet := pflag.NewFlagSet("", pflag.ExitOnError)
	flagSet.StringVarP(&intervalStr, "interval", "n", "2s", "seconds to wait between updates")
	flagSet.BoolVarP(&argument.isPrecise, "precise", "p", false, "attempt run command in precise intervals")
	flagSet.BoolVarP(&argument.isClockwork, "clockwork", "c", false, "run command in precise intervals forcibly")
	flagSet.BoolVar(&argument.isDebug, "debug", false, "")
	flagSet.BoolVarP(&argument.isDiff, "differences", "d", false, "highlight changes between updates")
	flagSet.BoolVarP(&argument.isNoTitle, "no-title", "t", false, "turn off header")
	flagSet.BoolVarP(&argument.isHelp, "help", "h", false, "display this help and exit")
	flagSet.BoolVarP(&argument.isVersion, "version", "v", false, "output version information and exit")
	flagSet.StringVar(&argument.shell, "shell", "sh", "shell (default \"sh\")")
	flagSet.StringVar(&argument.shellOpts, "shell-options", "", "additional shell options")

	flagSet.SetInterspersed(false)

	if err := flagSet.Parse(args); err != nil {
		return &argument, err
	}

	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		intervalFloat, err := strconv.ParseFloat(intervalStr, 64)
		if err != nil {
			return &argument, err
		}

		interval = time.Duration(intervalFloat * float64(time.Second))
	}

	argument.interval = interval

	if interval < 10*time.Millisecond {
		return nil, errIntervalTooSmall
	}

	rest := flagSet.Args()

	if len(rest) == 0 {
		return &argument, errNoCommand
	}

	argument.cmd = rest[0]
	argument.args = rest[1:]

	return &argument, nil
}

func help() {
	fmt.Println(`
Viddy well, gopher. Viddy well.

Usage:
 viddy [options] command

Options:
  -d, --differences          highlight changes between updates
  -n, --interval <interval>  seconds to wait between updates (default "2s")
  -p, --precise              attempt run command in precise intervals
  -c, --clockwork            run command in precise intervals forcibly
  -t, --no-title             turn off header
  --shell                    shell (default "sh")
  --shell-options            additional shell options

 -h, --help     display this help and exit
 -v, --version  output version information and exit`)
}
