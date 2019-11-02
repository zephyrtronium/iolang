package iolang

import "sync"

// Scheduler helps manage a group of Io coroutines.
type Scheduler struct {
	// m is a mutex to hold the scheduler's coros map.
	m sync.Mutex

	// Main is the first coroutine in the VM, instantiated with NewVM().
	Main *VM
	// Alive is a channel that closes once all coroutines stop.
	Alive <-chan bool

	// coros is the map of all active coroutines this scheduler manages, which
	// may include Main, to the VMs on which they are waiting, if any.
	coros map[*VM]*VM
	// start is a channel for coroutines to indicate that they are waiting on
	// another coroutine. This allows us to manage the dependency graph ourselves
	// so that we can detect deadlocks ahead of time and raise an exception,
	// instead of causing an unrecoverable panic if this is the only thing
	// happening in the Go program or staying in deadlock eternally if any other
	// goroutine is alive.
	start chan waitpair
	// pause is a channel for coroutines to indicate that they are pausing. This
	// removes it from coros as a key, but not as a value.
	pause chan *VM
	// finish is a channel for coroutines to indicate deactivation.
	finish chan *VM
}

// SchedulerTag is the Tag for the Scheduler.
const SchedulerTag = BasicTag("Scheduler")

// waitpair is a pair of coroutines such that a depends on b.
type waitpair struct {
	a, b *VM
}

func (vm *VM) initScheduler() {
	slots := Slots{
		"awaitingCoros": vm.NewCFunction(SchedulerAwaitingCoros, nil),
		"coroCount":     vm.NewCFunction(SchedulerCoroCount, nil),
		"type":          vm.NewString("Scheduler"),
		"yieldingCoros": vm.NewCFunction(SchedulerYieldingCoros, nil),
	}
	sched := &Scheduler{
		Main:   vm,
		coros:  map[*VM]*VM{vm: nil},
		start:  make(chan waitpair), // TODO: buffer?
		pause:  make(chan *VM),
		finish: make(chan *VM),
	}
	vm.Sched = sched
	vm.Core.SetSlot("Scheduler", &Object{
		Slots:  slots,
		Protos: []*Object{vm.BaseObject},
		Value:  sched,
		Tag:    SchedulerTag,
	})
	go sched.schedule()
}

// Start asks the scheduler to start a coroutine.
func (s *Scheduler) Start(coro *VM) {
	s.start <- waitpair{coro, nil}
}

// Await tells the scheduler that one coroutine is waiting on another.
func (s *Scheduler) Await(a, b *VM) {
	s.start <- waitpair{a, b}
}

// Finish tells the scheduler that a coroutine has finished execution.
func (s *Scheduler) Finish(coro *VM) {
	s.finish <- coro
}

// schedule manages the start, pause, and finish channels and detects deadlocks.
func (s *Scheduler) schedule() {
	alive := make(chan bool)
	defer close(alive)
	s.Alive = alive
	for len(s.coros) > 0 {
		select {
		case w := <-s.start:
			if w.b != nil {
				// Look for a cycle.
				s.m.Lock()
				if s.checkCycle(w) {
					s.m.Unlock()
					w.a.Control <- RemoteStop{w.a.NewExceptionf("deadlock"), ExceptionStop}
					continue
				}
				s.m.Unlock()
			}
			s.coros[w.a] = w.b
		case c := <-s.pause:
			s.m.Lock()
			delete(s.coros, c)
			s.m.Unlock()
		case c := <-s.finish:
			s.m.Lock()
			delete(s.coros, c)
			// Find all coroutines that depend on the coro which finished and
			// clear their dependencies.
			for a, b := range s.coros {
				if b == c {
					s.coros[a] = nil
				}
			}
			s.m.Unlock()
		}
	}
}

// checkCycle checks whether adding the given edge would form a cycle in the
// scheduler's dependency graph.
func (s *Scheduler) checkCycle(w waitpair) bool {
	for c := s.coros[w.b]; c != nil; c = s.coros[c] {
		if c == w.a {
			return true
		}
	}
	return false
}

// SchedulerAwaitingCoros is a Scheduler method.
//
// awaitingCoros returns a list of all coroutines which are waiting on another
// coroutine.
func SchedulerAwaitingCoros(vm *VM, target, locals *Object, msg *Message) *Object {
	var l []*Object
	vm.Sched.m.Lock()
	for a, b := range vm.Sched.coros {
		if b != nil {
			l = append(l, a.Coro)
		}
	}
	vm.Sched.m.Unlock()
	return vm.NewList(l...)
}

// SchedulerCoroCount is a Scheduler method.
//
// coroCount returns the number of active coroutines other than the current
// one. This is more efficient than using Scheduler yieldingCoros size.
func SchedulerCoroCount(vm *VM, target, locals *Object, msg *Message) *Object {
	vm.Sched.m.Lock()
	n := len(vm.Sched.coros) - 1
	vm.Sched.m.Unlock()
	return vm.NewNumber(float64(n))
}

// SchedulerYieldingCoros is a Scheduler method.
//
// yieldingCoros returns a list of all running coroutines except the current
// one.
func SchedulerYieldingCoros(vm *VM, target, locals *Object, msg *Message) *Object {
	var l []*Object
	vm.Sched.m.Lock()
	for a := range vm.Sched.coros {
		if a.Coro != vm.Coro {
			l = append(l, a.Coro)
		}
	}
	vm.Sched.m.Unlock()
	return vm.NewList(l...)
}
