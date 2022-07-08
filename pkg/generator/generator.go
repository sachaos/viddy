package generator

import (
	"context"
	"time"
)

type ClockGenerator struct {
	interval time.Duration
	out chan int64
}

func NewClockGenerator(interval time.Duration) *ClockGenerator {
	return &ClockGenerator{
		interval: interval,
		out: make(chan int64, 10),
	}
}

func (g *ClockGenerator) Run(ctx context.Context) {
	ticker := time.Tick(g.interval)
	begin := time.Now().UnixNano()

	go func() {
		for {
			select {
			case t := <- ticker:
				g.out <- (t.UnixNano() - begin) / int64(time.Millisecond)
			case <- ctx.Done():
				return
			}
		}
	}()
}

func (g *ClockGenerator) Out() <-chan int64 {
	return g.out
}

func (g *ClockGenerator) Done(id int64) {
	// do nothing
}

type PreciseGenerator struct {
	interval time.Duration
	out chan int64
	finish chan struct{}
}

func NewPreciseGenerator(interval time.Duration) *PreciseGenerator {
	return &PreciseGenerator{
		interval: interval,
		out: make(chan int64, 10),
		finish: make(chan struct{}),
	}
}

func (g *PreciseGenerator) Run(ctx context.Context) {
	go func() {
		begin := time.Now().UnixNano()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			t := time.Now()
			id := (t.UnixNano() - begin) / int64(time.Millisecond)

			g.out <- id

			<-g.finish

			pTime := time.Since(t)

			if pTime > g.interval {
				continue
			} else {
				time.Sleep(g.interval - pTime)
			}
		}
	}()
}

func (g *PreciseGenerator) Out() <-chan int64 {
	return g.out
}

func (g *PreciseGenerator) Done(id int64) {
	g.finish <- struct{}{}
}

type SequentialGenerator struct {
	interval time.Duration
	out chan int64
	finish chan struct{}
}

func NewSequentialGenerator(interval time.Duration) *SequentialGenerator {
	return &SequentialGenerator{
		interval: interval,
		out: make(chan int64, 10),
		finish: make(chan struct{}),
	}
}

func (g *SequentialGenerator) Run(ctx context.Context) {
	go func() {
		begin := time.Now().UnixNano()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			t := time.Now()
			id := (t.UnixNano() - begin) / int64(time.Millisecond)

			g.out <- id

			<-g.finish

			time.Sleep(g.interval)
		}
	}()
}

func (g *SequentialGenerator) Out() <-chan int64 {
	return g.out
}

func (g *SequentialGenerator) Done(id int64) {
	g.finish <- struct{}{}
}

