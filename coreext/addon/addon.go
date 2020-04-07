//go:generate go run ../../cmd/gencore addon_init.go addon ./io
//go:generate gofmt -s -w addon_init.go

package addon

import (
	"os"
	"plugin"
	"sync"

	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/internal"

	// importing for side effects
	_ "github.com/zephyrtronium/iolang/coreext/directory"
)

// Interface is the interface via which an addon is loaded into a VM.
//
// Addons in iolang are separate packages, which may be linked dynamically (on
// platforms supporting -buildmode=plugin) or statically. The addon is loaded
// by calling its IoAddon function, which must be of type func() Interface. This
// function will be called no more than once per interpreter. The plugin itself
// is opened only once per program, so its init functions may run less than
// once per time it is added to an interpreter.
//
// For dynamically loaded addons, the Io program's Importer uses a CFunction to
// lookup the IoAddon function in the plugin when needed. For statically
// linked addons, however, the program which creates the VM must manually load
// all addons using the VM's LoadAddon method.
type Interface = internal.Addon

func init() {
	internal.Register(initAddon)
}

func initAddon(vm *iolang.VM) {
	addonOnce.Do(initAddonOnce)
	internal.InitAddon(vm)
	slots := iolang.Slots{
		"havePlugins":      vm.IoBool(havePlugins),
		"open":             vm.NewCFunction(open, nil),
		"scanForNewAddons": vm.NewCFunction(scanForNewAddons, nil),
		"type":             vm.NewString("Addon"),
	}
	internal.CoreInstall(vm, "Addon", slots, nil, nil)
	internal.Ioz(vm, coreIo, coreFiles)
}

// addonOpen is an Addon method.
//
// open loads the addon at the receiver's path and returns the addon's object.
func open(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	p, proto := vm.GetSlot(target, "path")
	if proto == nil {
		return vm.RaiseExceptionf("addon path unset")
	}
	p.Lock()
	path, ok := p.Value.(iolang.Sequence)
	if !ok {
		p.Unlock()
		return vm.RaiseExceptionf("addon path must be Sequence, not %s", vm.TypeName(p))
	}
	plug, err := plugin.Open(path.String())
	p.Unlock()
	if err != nil {
		return vm.IoError(err)
	}
	open, err := plug.Lookup("IoAddon")
	if err != nil {
		return vm.IoError(err)
	}
	f, ok := open.(func() Interface)
	if !ok {
		return vm.RaiseExceptionf("%s is not an iolang addon", path)
	}
	<-vm.LoadAddon(f())
	return target
}

// addonScanForNewAddons is an Addon method.
//
// scanForNewAddons marks a directory to be scanned for new addons.
func scanForNewAddons(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	path, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	file, err := os.Open(path)
	if err != nil {
		return vm.IoError(err)
	}
	fi, err := file.Stat()
	if err != nil {
		return vm.IoError(err)
	}
	if !fi.IsDir() {
		return vm.RaiseExceptionf("%s is not a directory", path)
	}
	internal.ScanForAddons(vm, file)
	return target
}

// havePlugins indicates whether Go's plugin system is available on the current
// system. Currently this should become true on Linux or Darwin with cgo
// enabled, but the cgo requirement might drop in the future (unlikely) and more
// platforms might be added (likely).
var havePlugins = false

func initAddonOnce() {
	_, err := plugin.Open("/dev/null")
	if err == nil || err.Error() != "plugin: not implemented" {
		havePlugins = true
	}
}

var addonOnce sync.Once
