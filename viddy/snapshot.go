package viddy

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

// NewSnapshot returns Snapshot object.
//
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

func (s *Snapshot) prepareCommand(commands []string) *exec.Cmd {
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

	return command
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
		var err error
		if !s.diffPrepared {
			err = s.compareFromBefore()
		}

		if err == nil {
			src = DiffPrettyText(s.diff)
		}
	}

	if query != "" {
		src = strings.ReplaceAll(src, query, color.New(color.BgYellow, color.FgBlack).Sprintf(query))
	}

	src = tview.Escape(src)

	_, err := io.Copy(tview.ANSIWriter(w), strings.NewReader(src))

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
