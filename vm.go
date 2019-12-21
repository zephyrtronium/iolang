//go:generate go run ./cmd/gencore vm_init.go ./io
//go:generate gofmt -s -w vm_init.go

package iolang

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"time"
)

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

	// StartTime is the time at which VM initialization began, used for the
	// Date clock method.
	StartTime time.Time

	// Debug is an atomic flag controlling whether debugging is enabled for
	// this coroutine.
	Debug uint32
}

// NewVM prepares a new VM to interpret Io code. String arguments may be passed
// to occupy the System args slot, typically os.Args[1:].
func NewVM(args ...string) *VM {
	vm := VM{
		Lobby: &Object{Slots: Slots{}, id: nextObject()},

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
	// vm.initCall()
	vm.initMap()
	vm.initOpTable()
	vm.initObject()
	vm.initTrue()
	vm.initFalse()
	vm.initNil()
	vm.initLocals()
	vm.initList()
	vm.initFile()
	vm.initDirectory()
	vm.initDate()
	vm.initDuration()
	vm.initSystem()
	vm.initArgs(args)
	vm.initCollector()
	vm.initScheduler()
	vm.initCoroutine()
	vm.initFuture()
	vm.initAddon()
	vm.initPath()
	vm.initDebugger()

	vm.finalInit()

	return &vm
}

// coreInstall is a convenience method to install a new Core proto that has
// BaseObject as its proto.
func (vm *VM) coreInstall(proto string, slots Slots, value interface{}, tag Tag) {
	vm.Core.SetSlot(proto, vm.ObjectWith(slots, []*Object{vm.BaseObject}, value, tag))
}

// CoreProto returns a new Protos list for a type in vm.Core. Panics if there
// is no such type!
func (vm *VM) CoreProto(name string) []*Object {
	if p, ok := vm.Core.GetLocalSlot(name); ok {
		return []*Object{p}
	}
	panic("iolang: no Core proto named " + name)
}

// AddonProto returns a new Protos list for a type in vm.Addons. Panics if
// there is no such type!
func (vm *VM) AddonProto(name string) []*Object {
	if p, ok := vm.Addons.GetLocalSlot(name); ok {
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
	// Other init* functions will set up Core slots, but it is courteous to
	// make room for them.
	vm.Core.Slots = make(Slots, 46)
	vm.Core.Protos = []*Object{vm.BaseObject}
	vm.Addons.Protos = []*Object{vm.BaseObject}
	slots := Slots{"Core": vm.Core, "Addons": vm.Addons}
	protos := []*Object{vm.Core, vm.Addons}
	lp := vm.ObjectWith(slots, protos, nil, nil)
	vm.Lobby.Protos = []*Object{lp}
	vm.Lobby.Slots = Slots{"Protos": lp, "Lobby": vm.Lobby}
}

// finalInit runs Core initialization scripts once the VM can execute code.
func (vm *VM) finalInit() {
	for i, data := range coreIo {
		name := coreFiles[i]
		r, err := zlib.NewReader(bytes.NewReader(data))
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
