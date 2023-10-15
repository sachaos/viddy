package main

import "time"

type newSnapFunc func(int64, *Snapshot, chan<- struct{}) *Snapshot

func ClockSnapshot(begin int64, newSnap newSnapFunc, interval time.Duration) (<-chan *Snapshot, chan<- bool) {
	c := make(chan *Snapshot)
	isSuspended := false
	isSuspendedQueue := make(chan bool)

	go func() {
		var s *Snapshot

		t := time.Tick(interval)

		for now := range t {
			select {
			case isSuspended = <-isSuspendedQueue:
			default:
			}

			if isSuspended {
				continue
			}

			finish := make(chan struct{})
			id := (now.UnixNano() - begin) / int64(time.Millisecond)
			s = newSnap(id, s, finish)
			c <- s
		}
	}()

	return c, isSuspendedQueue
}

func PreciseSnapshot(newSnap newSnapFunc, interval time.Duration) (<-chan *Snapshot, chan<- bool) {
	c := make(chan *Snapshot)
	isSuspended := false
	isSuspendedQueue := make(chan bool)

	go func() {
		var s *Snapshot

		begin := time.Now().UnixNano()

		for {
			select {
			case isSuspended = <-isSuspendedQueue:
			default:
			}

			if isSuspended {
				time.Sleep(interval)

				continue
			}

			finish := make(chan struct{})
			start := time.Now()
			id := (start.UnixNano() - begin) / int64(time.Millisecond)
			ns := newSnap(id, s, finish)
			s = ns

			c <- ns

			<-finish

			pTime := time.Since(start)

			if pTime > interval {
				continue
			} else {
				time.Sleep(interval - pTime)
			}
		}
	}()

	return c, isSuspendedQueue
}

func SequentialSnapshot(newSnap newSnapFunc, interval time.Duration) (<-chan *Snapshot, chan<- bool) {
	c := make(chan *Snapshot)
	isSuspended := false
	isSuspendedQueue := make(chan bool)

	go func() {
		var s *Snapshot

		begin := time.Now().UnixNano()

		for {
			select {
			case isSuspended = <-isSuspendedQueue:
			default:
			}

			if isSuspended {
				time.Sleep(interval)

				continue
			}

			finish := make(chan struct{})
			id := (time.Now().UnixNano() - begin) / int64(time.Millisecond)
			s = newSnap(id, s, finish)
			c <- s

			<-finish

			time.Sleep(interval)
		}
	}()

	return c, isSuspendedQueue
}
