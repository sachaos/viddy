package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"
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
	NoCommand = errors.New("command is required")
	IntervalTooSmall = errors.New("interval too small")
)

func parseArguments(args []string) (*Arguments, error) {
	argument := Arguments{
		interval: 2 * time.Second,
	}
	var err error

LOOP:
	for len(args) != 0 {
		arg := args[0]
		args = args[1:]

		switch arg {
		case "-n", "--interval":
			if len(args) == 0 {
				return nil, errors.New("-n or --interval require argument")
			}
			interval := args[0]
			args = args[1:]
			argument.interval, err = time.ParseDuration(interval)
			if err != nil {
				seconds, err := strconv.Atoi(interval)
				if err != nil {
					return nil, err
				}
				argument.interval = time.Duration(seconds) * time.Second
			}
		case "-p", "--precise":
			argument.isPrecise = true
		case "-c", "--clockwork":
			argument.isClockwork = true
		case "--debug":
			argument.isDebug = true
		case "-d", "--differences":
			argument.isDiff = true
		case "-t", "--no-title":
			argument.isNoTitle = true
		case "-h", "--help":
			argument.isHelp = true
		case "-v", "--version":
			argument.isVersion = true
		default:
			args = append([]string{arg}, args...)
			break LOOP
		}
	}

	if len(args) == 0 {
		return &argument, NoCommand
	}

	if argument.interval < 10 * time.Millisecond {
		return nil, IntervalTooSmall
	}

	argument.cmd = args[0]
	argument.args = args[1:]

	return &argument, nil
}

func help() {
	fmt.Println(`
Viddy well, gopher. viddy well.

Usage:
 viddy [options] command

Options:
  -d, --differences          highlight changes between updates
  -n, --interval <interval>  seconds to wait between updates (default "2s")
  -p, --precise              attempt run command in precise intervals
  -c, --clockwork            run command in precise intervals forcibly
  -t, --no-title             turn off header

 -h, --help     display this help and exit
 -v, --version  output version information and exit`)
}