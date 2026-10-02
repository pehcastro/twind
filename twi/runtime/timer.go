package runtime

import (
	"slices"
	"time"
)

type Timer struct {
	rt *Runtime
	at time.Time
	fn func()
}

func (r *Runtime) After(d time.Duration, fn func()) *Timer {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now
	if !r.awake {
		now = r.cfg.Clock.Now()
		r.wakeUp()
	}
	t := &Timer{rt: r, at: now.Add(d), fn: fn}
	i := slices.IndexFunc(r.timers, func(o *Timer) bool { return o.at.After(t.at) })
	if i < 0 {
		i = len(r.timers)
	}
	r.timers = slices.Insert(r.timers, i, t)
	return t
}

func (t *Timer) Stop() {
	t.rt.mu.Lock()
	t.rt.timers = slices.DeleteFunc(t.rt.timers, func(o *Timer) bool { return o == t })
	t.rt.mu.Unlock()
}

func (r *Runtime) expire(now time.Time) {
	for {
		r.mu.Lock()
		if len(r.timers) == 0 || r.timers[0].at.After(now) {
			r.mu.Unlock()
			return
		}
		t := r.timers[0]
		r.timers = slices.Delete(r.timers, 0, 1)
		r.mu.Unlock()
		r.rebuild = true
		t.fn()
	}
}

func (r *Runtime) sleep() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.awake = false
	if len(r.timers) == 0 {
		return time.Time{}
	}
	return r.timers[0].at
}
