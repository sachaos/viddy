//go:build !windows

package run

import (
	"bytes"
	"context"
	"github.com/creack/pty"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func prepareCommand(ctx context.Context, commands []string, shell string, shellOpts string) *exec.Cmd {
	var command *exec.Cmd

	if runtime.GOOS == "windows" {
		cmdStr := strings.Join(commands, " ")
		compSec := os.Getenv("COMSPEC")
		command = exec.Command(compSec, "/c", cmdStr)
	} else {
		var args []string
		args = append(args, strings.Fields(shellOpts)...)
		args = append(args, "-c")
		args = append(args, strings.Join(commands, " "))
		command = exec.CommandContext(ctx, shell, args...) //nolint:gosec
	}

	return command
}

func runCommand(ctx context.Context, id int64, cmd string, args []string, shell, shellOpts string, width int, isPty bool, finished chan<- *Result) error {
	var b, eb bytes.Buffer

	commands := []string{cmd}
	commands = append(commands, args...)

	command := prepareCommand(ctx, commands, shell, shellOpts)
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
		r := Result{ID: id}
		if err := command.Wait(); err != nil {
			r.err = err
		}

		r.stdout = b.Bytes()
		r.stderr = eb.Bytes()
		r.exitCode = command.ProcessState.ExitCode()

		finished <- &r
	}()

	return nil
}
