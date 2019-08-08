package iolang

import (
	"plugin"
)

// Addon is an interface via which an addon is loaded into a VM.
//
// Addons in iolang are separate packages, which may be linked dynamically (on
// platforms supporting -buildmode=plugin) or statically. The addon is loaded by
// calling its OpenAddon function, which must be of type func(*VM) Addon.
//
// For dynamically loaded addons, the Io program's Importer uses a CFunction to
// lookup the OpenAddon function in the plugin when needed. For statically
// linked addons, however, the program which creates the VM must manually load
// all addons using the VM's LoadAddon method.
type Addon interface {
	// AddonName returns the addon's name, e.g. "Range".
	AddonName() string
	// Instance returns an object that can serve as a prototype in the VM's
	// Addons object. It is called exactly once each time a VM loads the addon.
	// This may be more than once in the lifetime of the plugin or program if
	// multiple separate VMs open it.
	Instance(vm *VM) Interface
	// Script returns a Message to be sent to the addon proto after it is
	// loaded. This allows an addon to compile an Io script to perform
	// additional setup once the proto exists. If the message is not nil, then
	// evaluating it must not raise an uncaught exception, otherwise the program
	// will panic.
	Script(vm *VM) *Message
}

func (vm *VM) initPlugin() {
	slots := Slots{
		"havePlugins": vm.IoBool(havePlugins),
		"open":        vm.NewCFunction(AddonOpen, nil),
		"type":        vm.NewString("Addon"),
	}
	vm.SetSlot(vm.Core, "Addon", vm.ObjectWith(slots))
}

// LoadAddon loads an addon. It returns a channel over which the addon object
// will be sent once it is loaded.
func (vm *VM) LoadAddon(addon Addon) chan Interface {
	ch := make(chan Interface, 1)
	vm.Sched.addons <- addontriple{vm, addon, ch}
	return ch
}

// reallyLoadAddon is the method the scheduler calls to set up an addon proto.
func (vm *VM) reallyLoadAddon(addon Addon) Interface {
	p := addon.Instance(vm)
	vm.SetSlot(vm.Addons, addon.AddonName(), p)
	m := addon.Script(vm)
	if m != nil {
		r, stop := m.Send(vm, p, p)
		if stop != NoStop {
			panic(r)
		}
	}
	return p
}

// AddonOpen is an Addon method.
//
// open loads the addon at the receiver's path and returns the addon's object.
func AddonOpen(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	p, proto := vm.GetSlot(target, "path")
	if proto == nil {
		return vm.RaiseException("addon path unset")
	}
	path, ok := p.(*Sequence)
	if !ok {
		return vm.RaiseExceptionf("addon path must be Sequence, not %s", vm.TypeName(p))
	}
	plug, err := plugin.Open(path.String())
	if err != nil {
		return vm.IoError(err)
	}
	open, err := plug.Lookup("OpenAddon")
	if err != nil {
		return vm.IoError(err)
	}
	f, ok := open.(func(*VM) Addon)
	if !ok {
		return vm.RaiseExceptionf("%s is not an iolang addon", path)
	}
	ch := vm.LoadAddon(f(vm))
	return <-ch, NoStop
}

// havePlugins indicates whether Go's plugin system is available on the current
// system. Currently this should become true on Linux or Darwin with cgo
// enabled, but the cgo requirement might drop in the future (unlikely) and more
// platforms might be added (likely).
var havePlugins = false

func init() {
	_, err := plugin.Open("/dev/null")
	if err == nil || err.Error() != "plugin: not implemented" {
		havePlugins = true
	}
}
