package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/fatih/color"
	"github.com/rivo/tview"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var dmp = diffmatchpatch.New()

type Snapshot struct {
	id int64

	command string
	args    []string

	shell     string
	shellOpts string

	result []byte
	start  time.Time
	end    time.Time

	exitCode    int
	errorResult []byte

	completed bool
	err       error

	diffPrepared bool
	diff         []diffmatchpatch.Diff

	diffAdditionCount int
	diffDeletionCount int

	before *Snapshot
	finish chan<- struct{}
}

//nolint:lll
func NewSnapshot(id int64, command string, args []string, shell string, shellOpts string, before *Snapshot, finish chan<- struct{}) *Snapshot {
	return &Snapshot{
		id:      id,
		command: command,
		args:    args,

		shell:     shell,
		shellOpts: shellOpts,

		before: before,
		finish: finish,
	}
}

func (s *Snapshot) compareFromBefore() error {
	if s.before != nil && !s.before.completed {
		return errNotCompletedYet
	}

	var beforeResult string
	if s.before == nil {
		beforeResult = ""
	} else {
		beforeResult = string(s.before.result)
	}

	s.diff = dmp.DiffCleanupSemantic(dmp.DiffMain(beforeResult, string(s.result), false))
	addition := 0
	deletion := 0

	for _, diff := range s.diff {
		//nolint:exhaustive
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			addition += len(diff.Text)
		case diffmatchpatch.DiffDelete:
			deletion += len(diff.Text)
		}
	}

	s.diffAdditionCount = addition
	s.diffDeletionCount = deletion
	s.diffPrepared = true

	return nil
}

//nolint:unparam
func (s *Snapshot) run(finishedQueue chan<- int64) error {
	s.start = time.Now()
	defer func() {
		s.end = time.Now()
	}()

	var b, eb bytes.Buffer

	commands := []string{s.command}
	commands = append(commands, s.args...)

	var command *exec.Cmd

	if runtime.GOOS == "windows" {
		cmdStr := strings.Join(commands, " ")
		compSec := os.Getenv("COMSPEC")
		command = exec.Command(compSec, "/c", cmdStr)
	} else {
		var args []string
		args = append(args, strings.Fields(s.shellOpts)...)
		args = append(args, "-c")
		args = append(args, strings.Join(commands, " "))
		command = exec.Command(s.shell, args...) //nolint:gosec
	}

	command.Stdout = &b
	command.Stderr = &eb

	if err := command.Start(); err != nil {
		return nil //nolint:nilerr
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

func isWhiteString(str string) bool {
	for _, c := range str {
		if !unicode.IsSpace(c) {
			return false
		}
	}

	return true
}

func (s *Snapshot) render(w io.Writer, isShowDiff bool, query string) error {
	src := string(s.result)

	if isWhiteString(src) {
		src = string(s.errorResult)
		_, err := io.WriteString(w, fmt.Sprintf(`[red]%s[-:-:-]`, src))

		return err
	}

	if isShowDiff {
		if s.diffPrepared {
			src = DiffPrettyText(s.diff)
		} else if err := s.compareFromBefore(); err == nil {
			src = DiffPrettyText(s.diff)
		}
	}

	var b bytes.Buffer
	if _, err := io.Copy(tview.ANSIWriter(&b), strings.NewReader(src)); err != nil {
		return err
	}

	var r io.Reader
	if query != "" {
		r = strings.NewReader(strings.ReplaceAll(b.String(), query, fmt.Sprintf(`[black:yellow]%s[-:-:-]`, query)))
	} else {
		r = &b
	}

	_, err := io.Copy(w, r)

	return err
}

func DiffPrettyText(diffs []diffmatchpatch.Diff) string {
	var buff bytes.Buffer

	for _, diff := range diffs {
		text := diff.Text

		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			for _, c := range text {
				if unicode.IsSpace(c) {
					_, _ = buff.WriteRune(c)
				} else {
					_, _ = buff.WriteString(color.New(color.BgGreen).Sprintf(string(c)))
				}
			}
		case diffmatchpatch.DiffEqual:
			_, _ = buff.WriteString(text)
		}
	}

	return buff.String()
}
