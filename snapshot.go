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

	"github.com/rivo/tview"

	"github.com/fatih/color"
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

	var b bytes.Buffer

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

	if err := command.Start(); err != nil {
		return nil //nolint:nilerr
	}

	go func() {
		if err := command.Wait(); err != nil {
			s.err = err
		}

		s.result = b.Bytes()
		s.completed = true
		finishedQueue <- s.id
		close(s.finish)
	}()

	return nil
}

func (s *Snapshot) render(w io.Writer, isShowDiff bool, query string) error {
	var err error

	var src string

	//nolint:nestif
	if isShowDiff {
		if s.diffPrepared {
			src = DiffPrettyText(s.diff)
		} else {
			err := s.compareFromBefore()
			if err != nil {
				src = string(s.result)
			} else {
				src = DiffPrettyText(s.diff)
			}
		}
	} else {
		src = string(s.result)
	}

	var b bytes.Buffer
	_, err = io.Copy(tview.ANSIWriter(&b), strings.NewReader(src))

	var r io.Reader
	if query != "" {
		r = strings.NewReader(strings.ReplaceAll(b.String(), query, fmt.Sprintf(`[:yellow]%s[-:-:-]`, query)))
	} else {
		r = &b
	}

	io.Copy(w, r)

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
