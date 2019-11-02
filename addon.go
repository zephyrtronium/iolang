package iolang

import (
	"os"
	"path/filepath"
	"plugin"
)

// Addon is an interface via which an addon is loaded into a VM.
//
// Addons in iolang are separate packages, which may be linked dynamically (on
// platforms supporting -buildmode=plugin) or statically. The addon is loaded
// by calling its IoAddon function, which must be of type func() Addon. This
// function will be called no more than once per interpreter. The plugin itself
// is opened only once per program, so its init functions may run less than
// once per time it is added to an interpreter.
//
// For dynamically loaded addons, the Io program's Importer uses a CFunction to
// lookup the IoAddon function in the plugin when needed. For statically
// linked addons, however, the program which creates the VM must manually load
// all addons using the VM's LoadAddon method.
type Addon interface {
	// Name returns the name of the addon.
	Name() string
	// Protos returns the list of protos this addon installs.
	Protos() []string
	// Depends returns the list of addons on which this addon depends. The VM
	// attempts to install each in the listed order before calling this addon's
	// Init. If any dependency cannot be installed, Init is never called.
	Depends() []string
	// Init initializes the plugin on this VM. For each proto the addon
	// provides, Init should call vm.Install with the base prototype.
	Init(vm *VM)
}

// addonmaps manages a VM's knowledge about available addons.
type addonmaps struct {
	// addons is a channel to synchronize loading addons.
	addons chan addontriple
	// scan channels paths to search for new addons.
	scan chan *os.File

	// known maps known addon names to their addons so the manager can load
	// dependencies.
	known map[string]Addon
	// opened tracks plugins that have already been opened.
	opened map[*plugin.Plugin]bool
	// inited tracks the names of addons that have already been initialized.
	inited map[string]bool

	// protos maps each proto name to the addon which provides it for the
	// importer.
	protos map[string]Addon
}

// addonplugin is a pair containing a plugin object and the addon initializer
// it provides.
type addonplugin struct {
	plug *plugin.Plugin
	f    func() Addon
}

// addontriple is a triple containing a coroutine waiting for an addon to be
// loaded, the addon it wants to load, and a channel to close once it loads.
type addontriple struct {
	coro *VM
	add  Addon
	ch   chan struct{}
}

func (vm *VM) manageAddons() {
	maps := vm.addonmaps
	for {
		select {
		case addon := <-maps.addons:
			name := addon.add.Name()
			if maps.inited[name] {
				continue
			}
			maps.inited[name] = true
			// Create a new coroutine to initialize the addon, in case it ends
			// up waiting on an existing coroutine.
			go func(c *VM, addon addontriple) {
				c.Sched.start <- waitpair{addon.coro, c}
				c.reallyLoadAddon(addon.add)
				c.Sched.finish <- c
				close(addon.ch)
			}(vm.VMFor(addon.coro.Coro.Clone()), addon)
		case p := <-maps.scan:
			for plug := range findAddons(p) {
				if maps.opened[plug.plug] {
					// The plugin is already open. Ignore it.
					continue
				}
				maps.opened[plug.plug] = true
				add := plug.f()
				maps.known[add.Name()] = add
				for _, proto := range add.Protos() {
					if _, ok := maps.protos[proto]; !ok {
						maps.protos[proto] = add
					}
				}
			}
		case <-vm.Sched.Alive:
			return
		}
	}
}

func (vm *VM) initAddon() {
	vm.addonmaps = &addonmaps{
		addons: make(chan addontriple),
		scan:   make(chan *os.File),
		protos: make(map[string]Addon),
		opened: make(map[*plugin.Plugin]bool),
		inited: make(map[string]bool),
		known:  make(map[string]Addon),
	}
	go vm.manageAddons()
	slots := Slots{
		"havePlugins": vm.IoBool(havePlugins),
		"open":        vm.NewCFunction(AddonOpen, nil),
		"type":        vm.NewString("Addon"),
	}
	vm.Core.SetSlot("Addon", vm.ObjectWith(slots))
}

// Install installs an addon proto by appending it to Lobby's protos and
// setting the corresponding slot in Addons.
func (vm *VM) Install(name string, proto *Object) {
	vm.Addons.SetSlot(name, proto)
	l := vm.Lobby
	l.Lock()
	l.Protos = append(l.Protos, proto)
	l.Unlock()
}

// LoadAddon loads an addon. It returns a channel that closes when the addon is
// loaded.
func (vm *VM) LoadAddon(addon Addon) <-chan struct{} {
	ch := make(chan struct{})
	vm.addonmaps.addons <- addontriple{vm, addon, ch}
	return ch
}

// reallyLoadAddon is the method the addon manager calls to set up an addon.
func (vm *VM) reallyLoadAddon(addon Addon) {
	for _, dep := range addon.Depends() {
		if vm.addonmaps.inited[dep] {
			continue
		}
		da, ok := vm.addonmaps.known[dep]
		if !ok {
			vm.RaiseExceptionf("unable to load %s (dependency of %s): not in any scanned directory", dep, addon.Name())
			return
		}
		<-vm.LoadAddon(da)
	}
	addon.Init(vm)
}

// findAddons yields addonplugins for Io addons in the given directory.
func findAddons(file *os.File) <-chan addonplugin {
	ch := make(chan addonplugin, 8)
	go func() {
		defer close(ch)
		defer file.Close()
		for {
			fis, err := file.Readdir(8)
			if err != nil {
				break
			}
			for _, fi := range fis {
				if fi.IsDir() {
					continue
				}
				path := filepath.Join(file.Name(), fi.Name())
				plug, err := plugin.Open(path)
				if err != nil {
					// TODO: maybe try to open as pure Io addon?
					continue
				}
				open, err := plug.Lookup("IoAddon")
				if err != nil {
					continue
				}
				f, ok := open.(func() Addon)
				if !ok {
					continue
				}
				ch <- addonplugin{plug, f}
			}
		}
	}()
	return ch
}

// AddonOpen is an Addon method.
//
// open loads the addon at the receiver's path and returns the addon's object.
func AddonOpen(vm *VM, target, locals *Object, msg *Message) *Object {
	p, proto := target.GetSlot("path")
	if proto == nil {
		return vm.RaiseExceptionf("addon path unset")
	}
	p.Lock()
	path, ok := p.Value.(Sequence)
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
	f, ok := open.(func(*VM) Addon)
	if !ok {
		return vm.RaiseExceptionf("%s is not an iolang addon", path)
	}
	<-vm.LoadAddon(f(vm))
	return target
}

// AddonScanForNewAddons is an Addon method.
//
// scanForNewAddons marks a directory to be scanned for new addons.
func AddonScanForNewAddons(vm *VM, target, locals *Object, msg *Message) *Object {
	path, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
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
	vm.addonmaps.scan <- file
	return target
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
