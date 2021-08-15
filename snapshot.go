package main

import (
	"bytes"
	"fmt"
	"github.com/fatih/color"
	"github.com/sergi/go-diff/diffmatchpatch"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/rivo/tview"
)

type Snapshot struct {
	id int64

	command string
	args    []string

	result []byte
	start  time.Time
	end    time.Time

	completed bool
	err       error

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
	if diff && s.before != nil && s.completed {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(string(s.before.result), string(s.result), false)
		src = DiffPrettyText(diffs)
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
