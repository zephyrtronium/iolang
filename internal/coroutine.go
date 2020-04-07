package internal

// A Coroutine holds control flow and debugging for a single Io coroutine.
type Coroutine struct {
	// Control is the control flow channel for the VM associated with this
	// coroutine.
	Control chan RemoteStop
	// Debug is a pointer to the VM's Debug flag.
	Debug *uint32
}

// RunCoro starts an inactive coroutine by activating its main slot. It should
// be used in a go statement.
func RunCoro(vm *VM) {
	vm.Perform(vm.Coro, vm.Coro, vm.IdentMessage("main"))
	vm.Sched.Finish(vm)
}

// run is a temporary proxy to RunCoro(vm).
func (vm *VM) run() {
	RunCoro(vm)
}

// tagCoro is the Tag type for Coroutine objects.
type tagCoro struct{}

func (tagCoro) Activate(vm *VM, self, target, locals, context *Object, msg *Message) *Object {
	return self
}

func (tagCoro) CloneValue(value interface{}) interface{} {
	return Coroutine{Control: make(chan RemoteStop, 1)}
}

func (tagCoro) String() string {
	return "Coroutine"
}

// CoroutineTag is the Tag for Coroutine objects. Activate returns the
// coroutine. CloneValue creates a new control flow channel and no debugging.
var CoroutineTag tagCoro

// VMFor creates a VM for a given Coroutine object so that it can run Io code.
// The new VM does not have debugging enabled. Panics if the object is not a
// Coroutine.
func (vm *VM) VMFor(coro *Object) *VM {
	coro.Lock()
	c := coro.Value.(Coroutine)
	r := &VM{
		Lobby:       vm.Lobby,
		Core:        vm.Core,
		Addons:      vm.Addons,
		BaseObject:  vm.BaseObject,
		True:        vm.True,
		False:       vm.False,
		Nil:         vm.Nil,
		Operators:   vm.Operators,
		Sched:       vm.Sched,
		Control:     c.Control,
		Coro:        coro,
		addonmaps:   vm.addonmaps,
		numberCache: vm.numberCache,
		StartTime:   vm.StartTime,
	}
	c.Debug = &r.Debug
	coro.Value = c
	coro.Unlock()
	return r
}

func (vm *VM) initCoroutine() {
	value := Coroutine{Control: vm.Control, Debug: &vm.Debug}
	vm.Coro = vm.ObjectWith(nil, []*Object{vm.BaseObject}, value, CoroutineTag)
}
