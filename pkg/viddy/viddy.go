package viddy

import (
	"context"
	"github.com/sachaos/viddy/pkg/config"
	"github.com/sachaos/viddy/pkg/generator"
	"github.com/sachaos/viddy/pkg/store"
	"github.com/sachaos/viddy/pkg/view"
	"github.com/sachaos/viddy/pkg/worker"
)

type Viddy struct {
	worker *worker.Worker
	generator generatorInterface
	store *store.Store
	view *view.View
}

func NewViddy(cmd string, args []string, options config.Options) *Viddy {
	var runHandler worker.RunFunc
	w := worker.NewWorker(options.Concurrency, cmd, args, options.Shell, options.ShellOpts, options.Width, options.IsPty, runHandler)
	v := &Viddy{
		worker: w,
	}

	switch options.Mode {
	case config.ViddyIntervalModeClockwork:
		v.generator = generator.NewClockGenerator(options.Interval)
	case config.ViddyIntervalModePrecise:
		v.generator = generator.NewPreciseGenerator(options.Interval)
	case config.ViddyIntervalModeSequential:
		v.generator = generator.NewSequentialGenerator(options.Interval)
	}

	v.view = view.NewView(options.Interval, cmd, args, view.HelpPage(options.KeyMapping))
	v.store = store.NewStore()

	return v
}

func (v *Viddy) Run(ctx context.Context) error {
	v.worker.Run(ctx)
	v.generator.Run(ctx)
	if err := v.view.Run(); err != nil {
		return err
	}

	for {
		select {
		case id := <- v.generator.Out():
			v.worker.In() <- &worker.Task{ID: id}
			v.store.Set(&store.Record{
				ID:     id,
			})
		case r := <- v.worker.Finished():
			v.generator.Done(r.ID)
			v.store.Set(&store.Record{
				ID:     r.ID,
				Result: r,
			})
		case <- ctx.Done():
			return nil
		}
	}
}

type generatorInterface interface {
	Run(ctx context.Context)
	Out() <- chan int64
	Done(id int64)
}
