// Go Timer
// This package provides two types of timers, a one time, and an interval.
// OneTimer[T any] is used for a simple timer callback after a certain period of time.
// IntervalTimer[T any] is used for a repetitive callback as close to the time interval as possible.
package gotimer

import (
	"fmt"
	"time"
)

type scheduler struct {
	timers []timer
	stop   chan bool
	queue  chan timer
}

type timer interface {
	getInterval() *time.Duration
	getCreated() time.Time
	tick(t time.Time)
	getStop() bool
}

var _scheduler *scheduler

func instance() *scheduler {
	if _scheduler == nil {
		_scheduler = &scheduler{
			timers: make([]timer, 0),
			stop:   make(chan bool),
			queue:  make(chan timer),
		}
		go _scheduler.poll()
	}
	return _scheduler
}

func (s *scheduler) poll() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("recovered from panic: %v", r)
		}
	}()
	b := false
	for {
		select {
		case <-s.stop:
			b = true
		case t := <-s.queue:
			s.timers = append(s.timers, t)
		default:
			del := make([]int, 0)
			for i, t := range s.timers {
				if t.getInterval() == nil {
					t.tick(time.Now()) // it's a one-timer
					del = append(del, i)
				} else {
					if t.getStop() {
						del = append(del, i)
					} else {
						t.tick(time.Now())
					}
				}
			}
			for _, idx := range del {
				s.timers = append(s.timers[:idx], s.timers[:idx+1]...)
			}
		}
		if b {
			break
		}
	}
	_scheduler = nil // delete the scheduler as we have exited our loop
}

func queue(t timer, delay *time.Duration) {
	s := instance()
	if delay != nil {
		go func() {
			time.Sleep(*delay)
			s.queue <- t
		}()
	} else {
		s.queue <- t
	}
}

type IntervalTimer[T any] struct {
	c  time.Time
	i  time.Duration
	n  time.Time
	fn func(T)
	t  T
	s  bool
}

func (self *IntervalTimer[T]) getCreated() time.Time {
	return self.c
}
func (self *IntervalTimer[T]) getInterval() *time.Duration {
	return &self.i
}
func (self *IntervalTimer[T]) getStop() bool {
	return self.s
}
func (self *IntervalTimer[T]) tick(t time.Time) {
	if t.After(self.n) {
		self.fn(self.t)
		self.n = t.Add(self.i)
	}
}

type OneTimer[T any] struct {
	c  time.Time
	fn func(T)
	t  T
}

func (self *OneTimer[T]) getCreated() time.Time {
	return self.c
}
func (self *OneTimer[T]) getInterval() *time.Duration {
	return nil
}
func (self *OneTimer[T]) getStop() bool {
	return true
}
func (self *OneTimer[T]) tick(t time.Time) {
	self.fn(self.t)
}

// NewIntervalTimer will create a repeating timer and call your supplied function `fn` with your supplied object `v` after `delay`
func NewIntervalTimer[T any](fn func(T), v T, delay *time.Duration, interval time.Duration) *IntervalTimer[T] {
	t := &IntervalTimer[T]{
		c:  time.Now().UTC(),
		i:  interval,
		n:  time.Now().Add(interval),
		t:  v,
		fn: fn,
	}
	defer queue(t, delay) // defer so we don't block returning
	return t
}

// NewOneTimer will create a single delayed function call to `fn` with your supplied object `v` after `delay`
func NewOneTimer[T any](fn func(T), v T, delay *time.Duration) *OneTimer[T] {
	t := &OneTimer[T]{
		c:  time.Now().UTC(),
		fn: fn,
		t:  v,
	}
	defer queue(t, delay) // defer so we don't block returning
	return t
}
