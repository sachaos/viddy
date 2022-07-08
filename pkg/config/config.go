package config

import (
	"github.com/gdamore/tcell/v2"
	"time"
)

type Options struct {
	Shell       string
	ShellOpts   string
	Concurrency int
	Width       int
	IsPty    bool
	Mode     ViddyIntervalMode
	Interval time.Duration

	KeyMapping KeyMapping

	// TODO: add some Options
}

type ViddyIntervalMode string

var (
	// ViddyIntervalModeClockwork is mode to run command in precise intervals forcibly.
	ViddyIntervalModeClockwork ViddyIntervalMode = "clockwork"

	// ViddyIntervalModePrecise is mode to run command in precise intervals like `watch -p`.
	ViddyIntervalModePrecise ViddyIntervalMode = "precise"

	// ViddyIntervalModeSequential is mode to run command sequential like `watch` command.
	ViddyIntervalModeSequential ViddyIntervalMode = "sequential"
)

var DefaultOptions = Options{
	Shell:       "/bin/sh",
	Concurrency: 10,
}

type KeyMapping struct {
	ToggleTimeMachine           map[KeyStroke]struct{}
	GoToPastOnTimeMachine       map[KeyStroke]struct{}
	GoToFutureOnTimeMachine     map[KeyStroke]struct{}
	GoToMorePastOnTimeMachine   map[KeyStroke]struct{}
	GoToMoreFutureOnTimeMachine map[KeyStroke]struct{}
	GoToNowOnTimeMachine        map[KeyStroke]struct{}
	GoToOldestOnTimeMachine     map[KeyStroke]struct{}
}

type KeyStroke struct {
	Key     tcell.Key
	Rune    rune
	ModMask tcell.ModMask
}

