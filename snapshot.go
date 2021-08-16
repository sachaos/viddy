package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/sergi/go-diff/diffmatchpatch"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/rivo/tview"
)

var dmp = diffmatchpatch.New()

type Snapshot struct {
	id int64

	command string
	args    []string

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

func NewSnapshot(id int64, command string, args []string, before *Snapshot, finish chan<- struct{}) *Snapshot {
	return &Snapshot{
		id:      id,
		command: command,
		args:    args,

		before: before,
		finish: finish,
	}
}

func (s *Snapshot) compareFromBefore() error {
	if s.before != nil && !s.before.completed {
		return errors.New("not completed")
	}

	var beforeResult string
	if s.before == nil {
		beforeResult = ""
	} else {
		beforeResult = string(s.before.result)
	}

	s.diff = dmp.DiffMain(beforeResult, string(s.result), false)
	addition := 0
	deletion := 0
	for _, diff := range s.diff {
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

func (s *Snapshot) run(finishedQueue chan<- int64) error {
	s.start = time.Now()
	defer func() {
		s.end = time.Now()
	}()

	var b bytes.Buffer

	command := exec.Command(s.command, s.args...)
	command.Stdout = &b

	if err := command.Start(); err != nil {
		return nil
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

func (s *Snapshot) render(w io.Writer, diff bool, query string) error {
	var err error
	var src string

	if diff && s.diffPrepared {
		src = DiffPrettyText(s.diff)
	} else {
		src = string(s.result)
	}

	if query != "" {
		src = strings.Replace(src, query, fmt.Sprintf(`["s"]%s[""]`, query), -1)
	}

	_, err = io.Copy(tview.ANSIWriter(w), strings.NewReader(src))
	return err
}

func DiffPrettyText(diffs []diffmatchpatch.Diff) string {
	var buff bytes.Buffer

	for _, diff := range diffs {
		text := diff.Text

		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			_, _ = buff.WriteString(color.New(color.BgGreen).Sprintf(text))
		case diffmatchpatch.DiffEqual:
			_, _ = buff.WriteString(text)
		}
	}

	return buff.String()
}
