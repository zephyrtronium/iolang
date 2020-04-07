//go:generate go run ../cmd/gencore vm_init.go internal ./io
//go:generate gofmt -s -w vm_init.go

package internal

import (
	"compress/zlib"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/zephyrtronium/contains"
)

// IoVersion is the interpreter version, used for the System version slot. It
// bears no relation to versions of the original implementation.
const IoVersion = "1"

// IoSpecVer is the Io language version, used for the System iospecVersion
// slot.
const IoSpecVer = "0.0.0"

// VM is an object for processing Io programs.
type VM struct {
	// Lobby is the default target of messages.
	Lobby *Object
	// Core is the object containing the basic prototypes of Io.
	Core *Object
	// Addons is the object containing imported addon protos.
	Addons *Object

	// Singletons.
	BaseObject *Object
	True       *Object
	False      *Object
	Nil        *Object
	Operators  *Object

	// protoSet is the set of protos checked during GetSlot.
	protoSet contains.Set
	// protoStack is the stack of protos to check during GetSlot.
	protoStack []*Object

	// Sched is the scheduler for this VM and all related coroutines.
	Sched *Scheduler
	// Control is a buffered channel for remote control of this coroutine. The
	// evaluator checks this between each message. NoStop stops tell the
	// coroutine to yield, and all other stops have their results returned.
	Control chan RemoteStop
	// Coro is the Coroutine object for this VM. The object's value contains the
	// Control channel and a pointer to Debug.
	Coro *Object

	// addonmaps manages the VM's knowledge of addons.
	addonmaps *addonmaps

	// numberCache is a list of cached Number objects.
	numberCache []*Object

	// StartTime is the time at which VM initialization began, used for the
	// Date clock method.
	StartTime time.Time

	// Debug is an atomic flag controlling whether debugging is enabled for
	// this coroutine.
	Debug uint32

	// ExitStatus is the (first) value passed to System exit. Only the
	// scheduler's main VM receives this value.
	ExitStatus int
}

// NewVM prepares a new VM to interpret Io code. String arguments may be passed
// to occupy the System args slot, typically os.Args[1:].
func NewVM(args ...string) *VM {
	haveVM = true // TODO: atomic?

	vm := VM{
		Lobby: &Object{id: nextObject()},

		Core:   &Object{id: nextObject()},
		Addons: &Object{id: nextObject()},

		BaseObject: &Object{id: nextObject()},
		True:       &Object{id: nextObject()},
		False:      &Object{id: nextObject()},
		Nil:        &Object{id: nextObject()},

		Control: make(chan RemoteStop, 1),

		StartTime: time.Now(),
	}

	// There is a specific order for initialization. First, we have to
	// initialize Core, so that other init methods can set up their protos on
	// it. Then, we must initialize CFunction, so that others can use
	// NewCFunction. Following that, we must initialize Sequence, which in turn
	// initializes String, so that we can use NewString. After that, we must
	// have Map before OpTable and OpTable before Object. Lastly, we must have
	// a scheduler in order to evaluate Io statements.
	vm.initCore()
	vm.initCFunction()
	vm.initSequence()
	vm.initMessage()
	vm.initNumber()
	vm.initException()
	vm.initBlock()
	vm.initCall()
	vm.initMap()
	vm.initOpTable()
	vm.initObject()
	vm.initTrue()
	vm.initFalse()
	vm.initNil()
	vm.initLocals()
	vm.initList()
	vm.initSystem()
	vm.initArgs(args)
	vm.initCoroutine()
	vm.initScheduler()

	vm.finalInit()

	return &vm
}

// coreInstall is a convenience method to install a new Core proto that has
// BaseObject as its proto. Returns the new proto.
func (vm *VM) coreInstall(proto string, slots Slots, value interface{}, tag Tag) *Object {
	return CoreInstall(vm, proto, slots, value, tag)
}

// CoreInstall is a transitional proxy to vm.coreInstall for core extensions.
func CoreInstall(vm *VM, proto string, slots Slots, value interface{}, tag Tag) *Object {
	r := vm.ObjectWith(slots, []*Object{vm.BaseObject}, value, tag)
	vm.SetSlot(vm.Core, proto, r)
	return r
}

// CoreProto returns a new Protos list for a type in vm.Core. Panics if there
// is no such type!
func (vm *VM) CoreProto(name string) []*Object {
	if p, ok := vm.GetLocalSlot(vm.Core, name); ok {
		return []*Object{p}
	}
	panic("iolang: no Core proto named " + name)
}

// AddonProto returns a new Protos list for a type in vm.Addons. Panics if
// there is no such type!
func (vm *VM) AddonProto(name string) []*Object {
	if p, ok := vm.GetLocalSlot(vm.Addons, name); ok {
		return []*Object{p}
	}
	panic("iolang: no Addons proto named " + name)
}

// IoBool converts a bool to the appropriate Io boolean object.
func (vm *VM) IoBool(c bool) *Object {
	if c {
		return vm.True
	}
	return vm.False
}

// AsBool attempts to convert an Io object to a bool by activating its
// asBoolean slot. If the object has no such slot, it is true.
func (vm *VM) AsBool(obj *Object) bool {
	if obj == nil {
		obj = vm.Nil
	}
	isTrue, _ := vm.Perform(obj, obj, vm.IdentMessage("asBoolean"))
	if isTrue == vm.False || isTrue == vm.Nil {
		return false
	}
	return true
}

// AsString attempts to convert an Io object to a string by activating its
// asString slot. If the object has no such slot but its Value is an
// fmt.Stringer, then it returns the value of String(); otherwise, a default
// representation is used. If the asString method raises an exception, then the
// exception message is the returned value.
func (vm *VM) AsString(obj *Object) string {
	if obj == nil {
		obj = vm.Nil
	}
	obj, _ = vm.Perform(obj, obj, vm.IdentMessage("asString"))
	if s, ok := obj.Value.(fmt.Stringer); ok {
		return s.String()
	}
	return fmt.Sprintf("%T_%p", obj, obj)
}

// initCore initializes Lobby, Core, and Addons for this VM. This only creates
// room for other init functions to work with.
func (vm *VM) initCore() {
	vm.Core.SetProtos(vm.BaseObject)
	vm.Addons.SetProtos(vm.BaseObject)
	slots := Slots{"Core": vm.Core, "Addons": vm.Addons}
	protos := []*Object{vm.Core, vm.Addons}
	lp := vm.ObjectWith(slots, protos, nil, nil)
	vm.Lobby.SetProtos(lp)
	vm.SetSlots(vm.Lobby, Slots{"Protos": lp, "Lobby": vm.Lobby})
}

// finalInit runs Core initialization scripts once the VM can execute code.
func (vm *VM) finalInit() {
	Ioz(vm, coreIo, coreFiles)
	for _, ext := range coreExt {
		ext(vm)
	}
}

// Ioz executes zlib-compressed Io scripts generated by gencore. Panics on any
// error.
func Ioz(vm *VM, io, names []string) {
	for i, data := range io {
		name := names[i]
		r, err := zlib.NewReader(strings.NewReader(data))
		if err != nil {
			panic(fmt.Errorf("iolang: error decompressing initialization code from %s: %w", name, err))
		}
		msg, err := vm.Parse(r, name)
		if err != nil {
			panic(fmt.Errorf("iolang: error parsing initialization code from %s: %w", name, err))
		}
		if result, stop := msg.Eval(vm, vm.Core); stop != NoStop {
			panic(fmt.Errorf("iolang: error executing initialization code from %s: %s (%v)", name, vm.AsString(result), stop))
		}
	}
}

// IsAlive returns true if the VM is alive. If this returns false, then the
// VM's scheduler has exited, and attempting to perform messages will panic.
func (vm *VM) IsAlive() bool {
	// Yield to make sure the scheduler has time to die if it's doing so.
	runtime.Gosched()
	select {
	case <-vm.Sched.Alive:
		return false
	default:
		return true
	}
}

// Register registers a core extension. Each function is called in the order it
// is registered; extensions that depend on other extensions need only import
// them. Register should be called from within init funcs. Panics if NewVM has
// been called.
func Register(f func(*VM)) {
	if haveVM {
		panic("iolang/internal: Register must be called before any VM is created")
	}
	coreExt = append(coreExt, f)
}

// coreExt is a list of core extensions that have been registered.
var coreExt = make([]func(*VM), 0, 10)

// haveVM becomes true once NewVM has been called.
var haveVM = false
