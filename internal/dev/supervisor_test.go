package dev

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
)

const patience = 5 * time.Second

type buildCall struct {
	ctx     context.Context
	changed []string
	reply   chan buildReply
}

type buildReply struct {
	exe string
	err error
}

type fakeChild struct {
	exe    string
	events *events
	done   chan error
	once   sync.Once
}

func (c *fakeChild) Stop() error {
	c.events.add("stop " + c.exe)
	c.once.Do(func() { close(c.done) })
	return nil
}

func (c *fakeChild) Done() <-chan error { return c.done }

type events struct {
	mu   sync.Mutex
	list []string
	errs []error
	seen chan struct{}
}

func (e *events) add(s string) {
	e.mu.Lock()
	e.list = append(e.list, s)
	e.mu.Unlock()
	e.seen <- struct{}{}
}

func (e *events) report(err error) {
	e.mu.Lock()
	e.errs = append(e.errs, err)
	e.mu.Unlock()
	e.seen <- struct{}{}
}

func (e *events) snapshot() ([]string, []error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.list), slices.Clone(e.errs)
}

type rig struct {
	t        *testing.T
	calls    chan buildCall
	changes  chan []string
	events   *events
	children map[string]*fakeChild
	cancel   context.CancelFunc
	ended    chan struct{}
	failNext error
}

func start(t *testing.T) *rig {
	r := &rig{t: t, calls: make(chan buildCall), changes: make(chan []string), events: &events{seen: make(chan struct{}, 64)}, children: map[string]*fakeChild{}, ended: make(chan struct{})}
	s := Supervisor{
		Build: func(ctx context.Context, changed []string) (string, error) {
			reply := make(chan buildReply)
			r.calls <- buildCall{ctx, changed, reply}
			got := <-reply
			return got.exe, got.err
		},
		Start: func(exe string) (Child, error) {
			if err := r.failNext; err != nil {
				r.failNext = nil
				r.events.add("failed start " + exe)
				return nil, err
			}
			c := &fakeChild{exe: exe, events: r.events, done: make(chan error, 1)}
			r.children[exe] = c
			r.events.add("start " + exe)
			return c, nil
		},
		Report: r.events.report,
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	go func() {
		s.Run(ctx, r.changes)
		close(r.ended)
	}()
	t.Cleanup(func() {
		cancel()
		<-r.ended
	})
	return r
}

func (r *rig) call() buildCall {
	r.t.Helper()
	select {
	case c := <-r.calls:
		return c
	case <-time.After(patience):
		r.t.Fatal("no build started")
		return buildCall{}
	}
}

func (r *rig) noCall() {
	r.t.Helper()
	select {
	case c := <-r.calls:
		r.t.Fatalf("a build started that should not have, changed %v", c.changed)
	case <-time.After(patience / 50):
	}
}

func (r *rig) change(files ...string) {
	r.t.Helper()
	select {
	case r.changes <- files:
	case <-time.After(patience):
		r.t.Fatal("the supervisor did not take a change")
	}
}

func (r *rig) settle(want int) {
	r.t.Helper()
	for range want {
		select {
		case <-r.events.seen:
		case <-time.After(patience):
			list, errs := r.events.snapshot()
			r.t.Fatalf("waited for %d events, have %v and errors %v", want, list, errs)
		}
	}
}

func (r *rig) expect(list []string, errs []error) {
	r.t.Helper()
	gotList, gotErrs := r.events.snapshot()
	if !slices.Equal(gotList, list) {
		r.t.Errorf("events %q, want %q", gotList, list)
	}
	if !slices.EqualFunc(gotErrs, errs, errors.Is) {
		r.t.Errorf("reports %v, want %v", gotErrs, errs)
	}
}

func (r *rig) runningA() {
	r.t.Helper()
	first := r.call()
	if first.changed != nil {
		r.t.Errorf("first build got changes %v, want none", first.changed)
	}
	first.reply <- buildReply{exe: "a"}
	r.settle(2)
}

func TestSupervisorChangeStartsOneBuild(t *testing.T) {
	r := start(t)
	r.runningA()
	r.noCall()
	r.change("x.go")
	c := r.call()
	if !slices.Equal(c.changed, []string{"x.go"}) {
		t.Errorf("build got changes %v, want [x.go]", c.changed)
	}
	r.noCall()
	c.reply <- buildReply{exe: "b"}
	r.settle(3)
	r.expect([]string{"start a", "stop a", "start b"}, []error{nil, nil})
}

func TestSupervisorChangeDuringBuildRestartsIt(t *testing.T) {
	r := start(t)
	r.runningA()
	r.change("x.go")
	first := r.call()
	r.change("y.go")
	select {
	case <-first.ctx.Done():
	case <-time.After(patience):
		t.Fatal("the running build was not cancelled by the second change")
	}
	r.noCall()
	first.reply <- buildReply{exe: "late"}
	second := r.call()
	if !slices.Equal(second.changed, []string{"x.go", "y.go"}) {
		t.Errorf("restarted build got changes %v, want [x.go y.go]", second.changed)
	}
	second.reply <- buildReply{exe: "b"}
	r.settle(3)
	r.noCall()
	r.expect([]string{"start a", "stop a", "start b"}, []error{nil, nil})
}

func TestSupervisorFailedBuildKeepsChild(t *testing.T) {
	broken := errors.New("input.go:32: undefined: x")
	r := start(t)
	r.runningA()
	r.change("x.go")
	r.call().reply <- buildReply{err: broken}
	r.settle(1)
	r.expect([]string{"start a"}, []error{nil, broken})
	r.change("x.go")
	c := r.call()
	if !slices.Equal(c.changed, []string{"x.go", "x.go"}) {
		t.Errorf("build after a failure got changes %v, want both saves", c.changed)
	}
	c.reply <- buildReply{exe: "b"}
	r.settle(3)
	r.expect([]string{"start a", "stop a", "start b"}, []error{nil, broken, nil})
}

func TestSupervisorFailedFirstBuildWaits(t *testing.T) {
	broken := errors.New("main.go:1: expected package")
	r := start(t)
	r.call().reply <- buildReply{err: broken}
	r.settle(1)
	select {
	case <-r.ended:
		t.Fatal("the supervisor ended after a failed first build")
	case <-time.After(patience / 50):
	}
	r.change("main.go")
	r.call().reply <- buildReply{exe: "a"}
	r.settle(2)
	r.expect([]string{"start a"}, []error{broken, nil})
}

func TestSupervisorFailedStartWaits(t *testing.T) {
	refused := errors.New("access denied")
	r := start(t)
	r.runningA()
	r.failNext = refused
	r.change("x.go")
	r.call().reply <- buildReply{exe: "b"}
	r.settle(3)
	r.expect([]string{"start a", "stop a", "failed start b"}, []error{nil, refused})
	r.change("x.go")
	r.call().reply <- buildReply{exe: "c"}
	r.settle(2)
	r.expect([]string{"start a", "stop a", "failed start b", "start c"}, []error{nil, refused, nil})
}

func TestSupervisorEndsWhenChildQuits(t *testing.T) {
	r := start(t)
	r.runningA()
	close(r.children["a"].done)
	select {
	case <-r.ended:
	case <-time.After(patience):
		t.Fatal("the supervisor kept running after its child quit")
	}
	r.expect([]string{"start a"}, []error{nil})
}

func TestSupervisorKeepsWatchingAfterCrash(t *testing.T) {
	crash := errors.New("exit status 2")
	r := start(t)
	r.runningA()
	r.children["a"].done <- crash
	r.settle(1)
	select {
	case <-r.ended:
		t.Fatal("the supervisor ended when its child crashed")
	case <-time.After(patience / 50):
	}
	r.change("x.go")
	r.call().reply <- buildReply{exe: "b"}
	r.settle(2)
	r.expect([]string{"start a", "start b"}, []error{nil, crash, nil})
}

func TestSupervisorCancelStopsChild(t *testing.T) {
	r := start(t)
	r.runningA()
	r.cancel()
	select {
	case <-r.ended:
	case <-time.After(patience):
		t.Fatal("the supervisor kept running after cancel")
	}
	r.settle(1)
	r.expect([]string{"start a", "stop a"}, []error{nil})
}
