package worker

import (
	"context"
	"github.com/sachaos/viddy/pkg/run"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestWorker(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		dummyRunFunc := func(ctx context.Context, id int64, cmd string, args []string, shell, shellOpts string, width int, isPty bool, finished chan<- *run.Result) error {
			finished <- &run.Result{
				ID:       id,
			}
			return nil
		}

		worker := NewWorker(1, "ls", []string{"-l"}, "test", "", 10, false, dummyRunFunc)

		ctx, cancel := context.WithCancel(context.Background())

		go worker.Run(ctx)

		worker.In() <- &Task{ID: 1234}

		finished := <- worker.Finished()

		assert.Equal(t, &run.Result{ID: 1234}, finished)

		cancel()
	})

	t.Run("run concurrently", func(t *testing.T) {
		dummyRunFunc := func(ctx context.Context, id int64, cmd string, args []string, shell, shellOpts string, width int, isPty bool, finished chan<- *run.Result) error {
			time.Sleep(1 * time.Second)
			finished <- &run.Result{
				ID:       id,
			}
			return nil
		}

		worker := NewWorker(3, "ls", []string{"-l"}, "test", "", 10, false, dummyRunFunc)

		ctx, cancel := context.WithTimeout(context.Background(), 2 * time.Second)

		go worker.Run(ctx)

		worker.In() <- &Task{ID: 1}
		worker.In() <- &Task{ID: 2}
		worker.In() <- &Task{ID: 3}

		var r []*run.Result

		r = append(r, <- worker.Finished())
		r = append(r, <- worker.Finished())
		r = append(r, <- worker.Finished())

		assert.Equal(t, 3, len(r))

		cancel()
	})
}
