package iolang

import (
	"sync"
)

// TODO: custom implementation of sync.Map to avoid interface conversions.
// Making it fast enough might require hijacking runtime hash functions.

// syncSlot is a synchronized slot. Once a particular VM accesses this slot,
// it becomes the slot's owner for the duration of the access, and other coros
// must wait until the owner releases it before they can access the slot.
type syncSlot struct {
	mu    sync.Mutex
	cond  sync.Cond // cond.L is &mu
	owner *VM
	holds int
	value *Object
}

// newSy creates a new syncSlot with the given value and one hold by vm, or
// with zero holds if vm is nil.
func newSy(vm *VM, value *Object) *syncSlot {
	sy := &syncSlot{
		value: value,
	}
	sy.cond.L = &sy.mu
	if vm != nil {
		sy.owner = vm
		sy.holds = 1
	}
	return sy
}

// claim causes vm to claim s, blocking until this becomes possible. Calling
// this with a nil VM puts the slot in an erroneous state.
func (s *syncSlot) claim(vm *VM) {
	s.mu.Lock()
	for s.owner != nil && s.owner != vm {
		// This uses s.cond.L, an interface, instead of s.mu directly, which
		// means it is much slower (not inlined). That's alright because this
		// is already the slow path.
		s.cond.Wait()
	}
	s.owner = vm
	s.holds++
	s.mu.Unlock()
}

// release releases one hold on the slot. If the number of holds reaches zero
// as a result, then the slot awakens one of its waiters. Calling this from a
// coroutine that is not the owner puts the slot in an erroneous state.
func (s *syncSlot) release() {
	s.mu.Lock()
	s.holds--
	if s.holds == 0 {
		s.owner = nil
		s.cond.Signal()
	}
	s.mu.Unlock()
}

// snap returns a snapshot of the value in s.
func (s *syncSlot) snap(vm *VM) *Object {
	s.claim(vm)
	r := s.value
	s.release()
	return r
}

// actualSlots is the type objects actually use for slots. Keys have type
// string and values have type *Object.
type actualSlots = sync.Map

// Slots holds the set of messages to which an object responds.
type Slots = map[string]*Object

// GetSlot checks obj and its ancestors in depth-first order without
// cycles for a slot, returning the slot value and the proto which had it.
// proto is nil if and only if the slot was not found. This method may acquire
// the object's lock, as well as the lock of each ancestor in turn.
func (vm *VM) GetSlot(obj *Object, slot string) (value, proto *Object) {
	if obj == nil {
		return nil, nil
	}
	// Check obj itself before using the graph traversal mechanisms.
	if sy := vm.localSyncSlot(obj, slot); sy != nil {
		value = sy.value
		sy.release()
		// The slot value can be nil if the value is currently being created,
		// like in x := x, or if the slot was created but then an exception
		// occurred while evaluating its result. In either case, the slot
		// doesn't actually exist, so it is correct to continue into ancestors.
		if value != nil {
			return value, obj
		}
	}
	sy, proto := vm.getSlotAncestor(obj, slot)
	if proto != nil {
		value = sy.value
		sy.release()
	}
	return
}

// GetSlotSync is like GetSlot, but it returns an extra function to synchronize
// accesses to the slot. Other VMs will block on attempts to read or write the
// same slot, whether it is on obj or an ancestor, until release is called.
// If it is not nil, release must be called exactly once. Both proto and
// release are nil if and only if the slot is not found.
func (vm *VM) GetSlotSync(obj *Object, slot string) (value, proto *Object, release func()) {
	if obj == nil {
		return nil, nil, nil
	}
	if sy := vm.localSyncSlot(obj, slot); sy != nil {
		// It'd be trivial to wrap sy.release in a *sync.Once so we don't have
		// to worry about multiple calls, but that would be slow and would mean
		// every call to this allocates.
		if sy.value != nil {
			return sy.value, obj, sy.release
		}
	}
	sy, proto := vm.getSlotAncestor(obj, slot)
	if proto != nil {
		value = sy.value
		release = sy.release
	}
	return
}

// getSlotAncestor finds a slot on obj's ancestors.
func (vm *VM) getSlotAncestor(obj *Object, slot string) (sy *syncSlot, proto *Object) {
	vm.protoSet.Reset()
	vm.protoSet.Add(obj.UniqueID())
	for {
		obj.Lock()
		switch {
		case obj.proto == nil:
			// The slot does not exist.
			obj.Unlock()
			return nil, nil
		case len(obj.plusproto) == 0:
			// One proto. We don't need to use vm.protoStack.
			rp := obj.proto
			obj.Unlock()
			obj = rp
			if sy := vm.localSyncSlot(obj, slot); sy != nil {
				return sy, obj
			}
			// Try again with the proto.
			if !vm.protoSet.Add(obj.UniqueID()) {
				return nil, nil
			}
		default:
			// Several protos. Using vm.protoStack is more efficient than using
			// the goroutine stack for explicit recursion.
			for i := len(obj.plusproto) - 1; i >= 0; i-- {
				if p := obj.plusproto[i]; vm.protoSet.Add(p.UniqueID()) {
					vm.protoStack = append(vm.protoStack, p)
				}
			}
			if vm.protoSet.Add(obj.proto.UniqueID()) {
				vm.protoStack = append(vm.protoStack, obj.proto)
			}
			obj.Unlock()
			if len(vm.protoStack) == 0 {
				// If all this object's protos have been checked already, stop.
				return nil, nil
			}
			for len(vm.protoStack) > 1 {
				obj = vm.protoStack[len(vm.protoStack)-1] // grab the top
				if sy := vm.localSyncSlot(obj, slot); sy != nil {
					if sy.value != nil {
						vm.protoStack = vm.protoStack[:0]
						return sy, obj
					}
					sy.release()
				}
				vm.protoStack = vm.protoStack[:len(vm.protoStack)-1] // actually pop
				obj.Lock()
				if obj.proto != nil {
					for i := len(obj.plusproto) - 1; i >= 0; i-- {
						if p := obj.plusproto[i]; vm.protoSet.Add(p.UniqueID()) {
							vm.protoStack = append(vm.protoStack, p)
						}
					}
					if vm.protoSet.Add(obj.proto.UniqueID()) {
						vm.protoStack = append(vm.protoStack, obj.proto)
					}
				}
				obj.Unlock()
			}
			// The stack is down to one object. Check it, then try to return to
			// faster cases.
			obj = vm.protoStack[0]
			vm.protoStack = vm.protoStack[:0]
			if sy := vm.localSyncSlot(obj, slot); sy != nil {
				return sy, obj
			}
		}
	}
}

// GetLocalSlot checks only obj's own slots for a slot.
func (vm *VM) GetLocalSlot(obj *Object, slot string) (value *Object, ok bool) {
	if obj == nil {
		return nil, false
	}
	if sy := vm.localSyncSlot(obj, slot); sy != nil {
		value = sy.value
		sy.release()
		if value == nil {
			return nil, false
		}
		return value, true
	}
	return nil, false
}

// GetLocalSlotSync is like GetLocalSlot, but it returns a function to
// synchronize accesses to the slot. Other VMs will block on attempts to read
// or write the same slot until release is called. If it is not nil, release
// must be called exactly once. release is nil if and only if the slot does not
// exist on obj.
func (vm *VM) GetLocalSlotSync(obj *Object, slot string) (value *Object, release func()) {
	if obj == nil {
		return nil, nil
	}
	sy := vm.localSyncSlot(obj, slot)
	if sy != nil && sy.value != nil {
		return sy.value, sy.release
	}
	return nil, nil
}

// localSyncSlot claims a slot if it exists on obj.
func (vm *VM) localSyncSlot(obj *Object, slot string) *syncSlot {
	s, ok := obj.slots.Load(slot)
	if ok {
		sy := s.(*syncSlot)
		sy.claim(vm)
		return sy
	}
	return nil
}

// GetAllSlots returns a copy of all slots on obj. This may block if another
// coroutine is accessing any slot on the object.
func (vm *VM) GetAllSlots(obj *Object) Slots {
	slots := Slots{}
	obj.slots.Range(func(key interface{}, value interface{}) bool {
		v := value.(*syncSlot).snap(vm)
		if v != nil {
			slots[key.(string)] = v
		}
		return true
	})
	return slots
}

// SetSlot sets the value of a slot on obj.
func (vm *VM) SetSlot(obj *Object, slot string, value *Object) {
	// See whether the slot already exists first so that we don't have to
	// allocate memory on every call.
	if s, ok := obj.slots.Load(slot); ok {
		sy := s.(*syncSlot)
		sy.claim(vm)
		sy.value = value
		sy.release()
		return
	}
	vm.newSlot(obj, slot, value).release()
}

// SetSlotSync creates a synchronized setter for a slot on obj. set must be
// called exactly once with the new slot value. Users should call SetSlotSync
// before evaluating the Io code that will determine the value, e.g.:
//
// 	set := vm.SetSlotSync(obj, slot)
// 	value, stop := msg.Eval(vm, locals)
// 	if stop != NoStop {
// 		set(nil)
// 		return vm.Stop(value, stop)
// 	}
// 	set(value)
//
// set must be called exactly once.
func (vm *VM) SetSlotSync(obj *Object, slot string) (set func(*Object)) {
	var sy *syncSlot
	// See whether the slot already exists first so that we don't have to
	// allocate memory on every call.
	if s, ok := obj.slots.Load(slot); ok {
		sy = s.(*syncSlot)
		sy.claim(vm)
	} else {
		sy = vm.newSlot(obj, slot, nil)
	}
	return func(value *Object) {
		sy.value = value
		sy.release()
	}
}

// newSlot creates a new slot on obj with the given value, or changes its value
// if it already exists.
func (vm *VM) newSlot(obj *Object, slot string, value *Object) *syncSlot {
	n := newSy(vm, value)
	if s, ok := obj.slots.LoadOrStore(slot, n); ok {
		// The slot was created between our earlier load and our attempt to
		// store just now.
		sy := s.(*syncSlot)
		sy.claim(vm)
		sy.value = value
		return sy
	}
	return n
}

// SetSlots sets the values of multiple slots on obj.
func (vm *VM) SetSlots(obj *Object, slots Slots) {
	for slot, value := range slots {
		vm.SetSlot(obj, slot, value)
	}
}

// definitelyNewSlots creates the given new slots on obj. This is intended for
// creating new objects; it is erroneous to use this if any slot in slots
// already exists on obj.
func (vm *VM) definitelyNewSlots(obj *Object, slots Slots) {
	for slot, value := range slots {
		obj.slots.Store(slot, newSy(nil, value))
	}
}

// RemoveSlot removes slots from obj's local slots, if they are present.
func (vm *VM) RemoveSlot(obj *Object, slots ...string) {
	for _, slot := range slots {
		obj.slots.Delete(slot)
	}
}
