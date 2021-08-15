package main

import (
	"errors"
	"fmt"
	"time"
)

type Arguments struct {
	interval  time.Duration
	isPrecise bool
	isActual  bool
	isDebug   bool
	isDiff    bool
	isNoTitle bool
	isHelp    bool
	isVersion bool

	cmd  string
	args []string
}

var NoCommand = errors.New("command is required")

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
				return nil, err
			}
		case "-p", "--precise":
			argument.isPrecise = true
		case "-a", "--actual":
			argument.isActual = true
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
  -a, --actual               run command in precise intervals forcibly
  -t, --no-title             turn off header

 -h, --help     display this help and exit
 -v, --version  output version information and exit`)
}