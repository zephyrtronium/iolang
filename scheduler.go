package iolang

import "sync"

// Scheduler helps manage a group of Io coroutines.
type Scheduler struct {
	Object

	// Main is the first coroutine in the VM, instantiated with NewVM().
	Main *VM

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
	// m is a mutex specific to coros.
	m sync.Mutex

	// addons is a channel to synchronize loading addons.
	addons chan addontriple
	// loaded is a set of the names of addons that have been loaded.
	loaded map[string]struct{}
}

// waitpair is a pair of coroutines such that a depends on b.
type waitpair struct {
	a, b *VM
}

// addontriple is a triple containing a coroutine waiting for an addon to be
// loaded, the addon it wants to load, and a channel over which to send the
// addon object once it loads.
type addontriple struct {
	coro *VM
	add  Addon
	ch   chan Interface
}

// Activate returns the scheduler.
func (s *Scheduler) Activate(vm *VM, target, locals, context Interface, msg *Message) Interface {
	return s
}

// Clone returns the scheduler. In this implementation, the scheduler is a
// singleton; actual clones are not allowed.
func (s *Scheduler) Clone() Interface {
	return s
}

func (vm *VM) initScheduler() {
	slots := Slots{
		"type":          vm.NewString("Scheduler"),
		"yieldingCoros": vm.NewCFunction(SchedulerYieldingCoros),
	}
	sched := &Scheduler{
		Object: Object{Slots: slots, Protos: []Interface{vm.BaseObject}},
		Main:   vm,
		coros:  map[*VM]*VM{vm: nil},
		start:  make(chan waitpair), // TODO: buffer?
		pause:  make(chan *VM),
		finish: make(chan *VM),
		addons: make(chan addontriple),
		loaded: make(map[string]struct{}),
	}
	vm.Sched = sched
	SetSlot(vm.Core, "Scheduler", sched)
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
loop:
	for len(s.coros) > 0 {
		select {
		case w := <-s.start:
			if w.b != nil {
				// Look for a cycle.
				s.m.Lock()
				for c := s.coros[w.b]; c != nil; c = s.coros[c] {
					if c == w.a {
						s.m.Unlock()
						w.a.Stop <- w.a.RaiseException("deadlock").(Stop)
						continue loop
					}
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
		case a := <-s.addons:
			if _, ok := s.loaded[a.add.AddonName()]; ok {
				continue
			}
			s.loaded[a.add.AddonName()] = struct{}{}
			// Create a new coroutine to run the init script for the addon, in
			// case it ends up waiting on Main.
			c := a.coro.Clone().(*VM)
			go func() {
				s.start <- waitpair{a.coro, c}
				s.finish <- c
				a.ch <- c.reallyLoadAddon(a.add)
				close(a.ch)
			}()
		}
	}
}

// SchedulerYieldingCoros is a Scheduler method.
//
// yieldingCoros returns a list of all coroutines which are waiting on another
// coroutine.
func SchedulerYieldingCoros(vm *VM, target, locals Interface, msg *Message) Interface {
	var l []Interface
	vm.Sched.m.Lock()
	for a, b := range vm.Sched.coros {
		if b != nil {
			l = append(l, a)
		}
	}
	vm.Sched.m.Unlock()
	return vm.NewList(l...)
}
