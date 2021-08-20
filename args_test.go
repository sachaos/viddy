package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_parseArguments(t *testing.T) {
	testCases := []struct {
		name   string
		args   []string
		exp    *Arguments
		expErr error
	}{
		{
			name: "-n 2s ls -l",
			args: []string{"-n", "2s", "ls", "-l"},
			exp: &Arguments{
				interval:    2 * time.Second,
				isPrecise:   false,
				isClockwork: false,
				cmd:         "ls",
				args:        []string{"-l"},
			},
		},
		{
			name: "-n 1 ls -l",
			args: []string{"-n", "1", "ls", "-l"},
			exp: &Arguments{
				interval:    1 * time.Second,
				isPrecise:   false,
				isClockwork: false,
				cmd:         "ls",
				args:        []string{"-l"},
			},
		},
		{
			name:   "invalid interval",
			args:   []string{"-n", "1ms", "ls", "-l"},
			exp:    nil,
			expErr: IntervalTooSmall,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			argument, _, err := parseArguments(tt.args)
			assert.Equal(t, tt.expErr, err)
			assert.Equal(t, tt.exp, argument)
		})
	}
}
