//go:build !windows

package main

import (
	"bytes"
	"github.com/creack/pty"
	"io"
)

func (s *Snapshot) run(finishedQueue chan<- int64, width int, isPty bool) error {
	var b, eb bytes.Buffer

	commands := []string{s.command}
	commands = append(commands, s.args...)

	command := s.prepareCommand(commands)
	command.Stderr = &eb

	if isPty {
		pty, err := pty.StartWithSize(command, &pty.Winsize{
			Cols: uint16(width),
		})
		if err != nil {
			return err
		}

		go func() {
			_, _ = io.Copy(&b, pty)
		}()
	} else {
		command.Stdout = &b
		if err := command.Start(); err != nil {
			return err
		}
	}

	go func() {
		if err := command.Wait(); err != nil {
			s.err = err
		}

		s.result = b.Bytes()
		s.errorResult = eb.Bytes()
		s.exitCode = command.ProcessState.ExitCode()
		s.completed = true
		finishedQueue <- s.id
		close(s.finish)
	}()

	return nil
}
