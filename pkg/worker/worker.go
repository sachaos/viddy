package worker

import (
	"context"
	"github.com/sachaos/viddy/pkg/run"
	"log"
)

type Task struct {
	ID int64
}

type RunFunc func(ctx context.Context, id int64, cmd string, args []string, shell, shellOpts string, width int, isPty bool, finished chan<- *run.Result) error

type Worker struct {
	queue       chan *Task
	finished    chan *run.Result
	concurrency int

	cmd       string
	args      []string
	shell     string
	shellOpts string
	width     int
	isPty     bool

	runHandler RunFunc
}

func NewWorker(concurrency int, cmd string, args []string, shell, shellOpts string, width int, isPty bool, runHandler RunFunc) *Worker {
	return &Worker{
		queue:       make(chan *Task, 10),
		finished:    make(chan *run.Result, 10),
		concurrency: concurrency,

		cmd:       cmd,
		args:      args,
		shell:     shell,
		shellOpts: shellOpts,
		width:     width,
		isPty:     isPty,

		runHandler: runHandler,
	}
}

func (w *Worker) Run(ctx context.Context) {
	finished := make(chan *run.Result, 10)

	go func() {
		for {
			select {
			case r := <- finished:
				w.finished <- r
			case <- ctx.Done():
				close(w.finished)
				close(w.queue)
				return
			}
		}
	}()

	for i := 0; i < w.concurrency; i++ {
		go func() {
			ctx2, cancel := context.WithCancel(ctx)
			for {
				select {
				case t := <-w.queue:
					if t == nil {
						continue
					}

					if err := w.runHandler(ctx2, t.ID, w.cmd, w.args, w.shell, w.shellOpts, w.width, w.isPty, finished); err != nil {
						log.Println(err)
					}
				case <- ctx.Done():
					cancel()
				}
			}
		}()
	}
}

func (w *Worker) In() chan<- *Task {
	return w.queue
}

func (w *Worker) Finished() <-chan *run.Result {
	return w.finished
}
