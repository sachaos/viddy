package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

//nolint:funlen
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
				shell:       "sh",
				args: []string{"-l"},
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
				shell:       "sh",
				args:        []string{"-l"},
			},
		},
		{
			name: "-n1 tail -n 1 hoge",
			args: []string{"-n1", "tail", "-n", "1", "hoge"},
			exp: &Arguments{
				interval:    1 * time.Second,
				isPrecise:   false,
				isClockwork: false,
				cmd:         "tail",
				shell:       "sh",
				args:        []string{"-n", "1", "hoge"},
			},
		},
		{
			name: "tail -n 1 hoge",
			args: []string{"tail", "-n", "1", "hoge"},
			exp: &Arguments{
				interval:    2 * time.Second,
				isPrecise:   false,
				isClockwork: false,
				cmd:         "tail",
				shell:       "sh",
				args:        []string{"-n", "1", "hoge"},
			},
		},
		{
			name: "-n 0.5 ls",
			args: []string{"-n", "0.5", "ls"},
			exp: &Arguments{
				interval:    500 * time.Millisecond,
				isPrecise:   false,
				isClockwork: false,
				cmd:         "ls",
				shell:       "sh",
				args:        []string{},
			},
		},
		{
			name:   "invalid interval",
			args:   []string{"-n", "1ms", "ls", "-l"},
			exp:    nil,
			expErr: errIntervalTooSmall,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			argument, err := parseArguments(tt.args)
			assert.Equal(t, tt.expErr, err)
			assert.Equal(t, tt.exp, argument)
		})
	}
}
