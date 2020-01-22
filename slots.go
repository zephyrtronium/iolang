package iolang

import (
	"math/bits"
	"sync"
	"sync/atomic"
	"unsafe"
)

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

// actualSlots is a synchronized trie structure that implements slots.
//
// The trie is grow-only. A leaf value of nil indicates an empty slot that has
// not been created; a slot value of nil indicates an unset or deleted slot.
type actualSlots struct {
	root *slotRecord // atomic
}

// slotRecord represents one piece of the slots trie. Its fields must be
// manipulated atomically.
type slotRecord struct {
	// leaf is the value associated with the string ending at the current node.
	leaf *syncSlot
	// scut is a read-only shortcut to the leaf that justified creating this
	// branch, if there is one. It does not need to be accessed atomically.
	scut *syncSlot
	// scutName is the name of the shortcut slot beginning at this node's
	// parent edge value. It does not need to be accessed atomically.
	scutName string
	// mask is the list of the names of this record's child nodes. The first
	// sibling always has 00 as its first entry; otherwise, a zero byte
	// indicates no edge.
	mask uintptr
	// children is this record's child nodes.
	children [recordChildren]*slotRecord
	// sibling is the next record at the same level in the trie.
	sibling *slotRecord
}

// recordChildren is the number of children in a single slotRecord, i.e. the
// size of uintptr in bytes.
const recordChildren = 4 << (^uintptr(0) >> 32 & 1)

// recordPop has the first bit in each byte set and all others clear.
const recordPop = ^uintptr(0) / 0xff

// load finds the given slot, or returns nil if there is no such slot.
func (s *actualSlots) load(slot string) *syncSlot {
	cur := (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.root))))
	if cur == nil {
		return nil
	}
	// Iterate manually rather than by range; we want bytes, not runes.
	for i := 0; i < len(slot); i++ {
		c := slot[i]
		if c == 0 {
			// The nul slot is always the first child of the first record on
			// the current level, even if it doesn't exist.
			cur = (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.children[0]))))
			if cur == nil {
				return nil
			}
			continue
		}
		m := uintptr(c) * recordPop
		for {
			if cur == nil {
				return nil
			}
			// Check the shortcut.
			if cur.scut != nil && cur.scutName == slot[i:] {
				return cur.scut
			}
			// From https://graphics.stanford.edu/~seander/bithacks.html
			// v has a zero byte iff that byte in mask is equal to c.
			v := atomic.LoadUintptr(&cur.mask) ^ m
			// After subtracting recordPop, the high bit of each byte in v is
			// set iff the byte either was 0 or was greater than 0x80. After
			// clearing the bits that were set in v, the latter case is
			// eliminated. Then, masking the high bit in each byte leaves the
			// bits whose bytes contained c as the only ones still set.
			v = (v - recordPop) &^ v & (recordPop << 7)
			if v == 0 {
				// No match in this record.
				cur = (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.sibling))))
				continue
			}
			// The lowest set bit is in the byte that matched c.
			// There isn't currently any platform where uint is smaller than
			// uintptr, but this check happens at compile-time anyway.
			var k int
			if uint64(^uint(0)) >= uint64(^uintptr(0)) {
				k = bits.TrailingZeros(uint(v)) / 8
			} else {
				k = bits.TrailingZeros64(uint64(v)) / 8
			}
			next := (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.children[k]))))
			for next == nil {
				// The edge to this child exists, but the node hasn't been set.
				// Another goroutine must be in the process of setting it. Spin
				// until it's available.
				next = (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.children[k]))))
			}
			cur = next
			break
		}
	}
	return (*syncSlot)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.leaf))))
}

// open loads the current value of the given slot if it exists or creates a new
// slot there if it does not. The slot is claimed by vm.
func (s *actualSlots) open(vm *VM, slot string) *syncSlot {
	cur := (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.root))))
	if cur == nil {
		// Slots are empty. Try to create them, but it's possible we're not the
		// only ones doing so; whoever gets there first wins.
		cur = &slotRecord{}
		if !atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&s.root)), nil, unsafe.Pointer(cur)) {
			cur = (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.root))))
		}
	}
	// Iterate manually rather than by range; we want bytes, not runes.
	for i := 0; i < len(slot); i++ {
		c := slot[i]
		if c == 0 {
			// The nul slot is always the first child of the first record on
			// the current level, even if it doesn't exist.
			cur = &slotRecord{}
			if !atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.children[0])), nil, unsafe.Pointer(cur)) {
				cur = (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.children[0]))))
			}
			continue
		}
		m := uintptr(c) * recordPop
		first := uintptr(1)
		for {
			cm := atomic.LoadUintptr(&cur.mask)
			// From https://graphics.stanford.edu/~seander/bithacks.html
			// v has a zero byte iff that byte in mask is equal to c.
			v := cm ^ m
			// After subtracting recordPop, the high bit of each byte in v is
			// set iff the byte either was 0 or was greater than 0x80. After
			// clearing the bits that were set in v, the latter case is
			// eliminated. Then, masking the high bit in each byte leaves the
			// bits whose bytes contained c as the only ones still set.
			v = (v - recordPop) &^ v & (recordPop << 7)
			if v == 0 {
				// No match in this record.
				if cm < 1<<(32<<(^uintptr(0)>>32&1)-7) {
					// This record had at least one open spot at the time we
					// loaded its mask. Locate it, create a new mask, and try
					// to commit it to reserve the edge. The technique here is
					// essentially the same as above (and below).
					v = (cm | first - recordPop) &^ (cm | first) & (recordPop << 7)
					var k int
					if uint64(^uint(0)) >= uint64(^uintptr(0)) {
						k = bits.TrailingZeros(uint(v)) / 8
					} else {
						k = bits.TrailingZeros(uint(v)) / 8
					}
					n := cm | uintptr(c)<<(k*8)
					if atomic.CompareAndSwapUintptr(&cur.mask, cm, n) {
						// We got the edge. Now we can allocate the node, as
						// well as the entire rest of the branch.
						node, leaf := recordBranch(vm, slot[i+1:])
						atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&cur.children[k])), unsafe.Pointer(node))
						return leaf
					}
					// Someone else added an edge, and it might have been the
					// same edge we're looking for. Try again from the start.
					continue
				}
				first = 0
				next := (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.sibling))))
				if next == nil {
					// This was the last record in this level. Try to add a new
					// sibling. Another goroutine might create a new sibling
					// while we're allocating ours, though, so we don't want to
					// try to make the whole branch.
					next = &slotRecord{}
					if !atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.sibling)), nil, unsafe.Pointer(next)) {
						// Someone else added the sibling. Use theirs.
						next = (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.sibling))))
					}
				}
				cur = next
				continue
			}
			// The lowest set bit is in the byte that matched c. There isn't
			// currently any platform where uint is smaller than uintptr, but
			// this check happens at compile-time anyway.
			var k int
			if uint64(^uint(0)) >= uint64(^uintptr(0)) {
				k = bits.TrailingZeros(uint(v)) / 8
			} else {
				k = bits.TrailingZeros64(uint64(v)) / 8
			}
			next := (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.children[k]))))
			for next == nil {
				// The edge to this child exists, but the node hasn't been set.
				// Another goroutine must be in the process of setting it. Spin
				// until it's available.
				next = (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.children[k]))))
			}
			cur = next
			break
		}
	}
	sy := (*syncSlot)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.leaf))))
	if sy == nil {
		// The node existed, but its value was unset. This happens if the slot
		// we're creating is a prefix of a slot that was added earlier.
		sy = newSy(vm, nil)
		if atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.leaf)), nil, unsafe.Pointer(sy)) {
			return sy
		}
		// Someone else created the slot before us. Use theirs.
		sy = (*syncSlot)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.leaf))))
	}
	sy.claim(vm)
	return sy
}

// foreach executes a function for each slot. If exec returns false, then the
// iteration ceases. The slots passed to exec are claimed by VM and will be
// released afterward.
func (s *actualSlots) foreach(vm *VM, exec func(name string, sy *syncSlot) bool) {
	cur := (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.root))))
	if cur == nil {
		return
	}
	cur.foreachIter(vm, exec, nil)
}

// foreachIter executes actualSlots.foreach for a single depth of the trie.
func (r *slotRecord) foreachIter(vm *VM, exec func(name string, sy *syncSlot) bool, b []byte) []byte {
	sy := (*syncSlot)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&r.leaf))))
	if sy != nil {
		r := true
		sy.claim(vm)
		if sy.value != nil {
			r = exec(string(b), sy)
		}
		sy.release()
		if !r {
			return nil
		}
	}
	// Handle the first record specially, since it contains the zero edge.
	cur := (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&r.children[0]))))
	if cur != nil {
		b = append(b, 0)
		b = cur.foreachIter(vm, exec, b)
		if b == nil {
			return nil
		}
		b = b[:len(b)-1]
	}
	for k := 1; k < recordChildren; k++ {
		cm := atomic.LoadUintptr(&r.mask)
		c := byte(cm >> (k * 8))
		if c == 0 {
			// No more edges.
			return b
		}
		cur = (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&r.children[k]))))
		if cur == nil {
			// The node is being created. Even if we wait for it, it won't have
			// a real value until later. Furthermore, if we were to try to spin
			// for it, then trying to create a slot from within foreach would
			// have the potential to enter an active deadlock. Just skip.
			continue
		}
		b = append(b, c)
		b = cur.foreachIter(vm, exec, b)
		if b == nil {
			return nil
		}
		b = b[:len(b)-1]
	}
	// Now we can loop over siblings.
	for {
		r = (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&r.sibling))))
		if r == nil {
			// Last sibling. We're done.
			return b
		}
		for k := 0; k < recordChildren; k++ {
			cm := atomic.LoadUintptr(&r.mask)
			c := byte(cm >> (k * 8))
			if c == 0 {
				return b
			}
			cur = (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&r.children[k]))))
			if cur == nil {
				continue
			}
			b = append(b, c)
			b = cur.foreachIter(vm, exec, b)
			if b == nil {
				return nil
			}
			b = b[:len(b)-1]
		}
	}
}

// recordBranch allocates an entire new branch of a slots trie.
func recordBranch(vm *VM, branch string) (trunk *slotRecord, slot *syncSlot) {
	trunk = &slotRecord{}
	cur := trunk
	for i := 0; i < len(branch); i++ {
		cur.mask = uintptr(branch[i]) << 8
		cur.children[1] = &slotRecord{}
		cur = cur.children[1]
	}
	cur.leaf = newSy(vm, nil)
	if len(branch) > 1 {
		// Only create shortcuts that actually skip nodes.
		trunk.scut = cur.leaf
		trunk.scutName = branch
	}
	return trunk, cur.leaf
}

// Slots represents the set of messages to which an object responds.
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
		return value, obj
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
		return sy.value, obj, sy.release
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
			if !vm.protoSet.Add(obj.UniqueID()) {
				return nil, nil
			}
			if sy := vm.localSyncSlot(obj, slot); sy != nil {
				return sy, obj
			}
			// Try again with the proto.
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
					vm.protoStack = vm.protoStack[:0]
					return sy, obj
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
	if sy != nil {
		return sy.value, sy.release
	}
	return nil, nil
}

// localSyncSlot claims a slot if it exists on obj.
func (vm *VM) localSyncSlot(obj *Object, slot string) *syncSlot {
	sy := obj.slots.load(slot)
	if sy == nil {
		return nil
	}
	sy.claim(vm)
	if sy.value == nil {
		// The slot value can be nil if the value is currently being created,
		// like in x := x, or if the slot was created but then an exception
		// occurred while evaluating its result, or if the slot was removed. In
		// any case, the slot doesn't actually exist.
		sy.release()
		return nil
	}
	return sy
}

// GetAllSlots returns a copy of all slots on obj. This may block if another
// coroutine is accessing any slot on the object.
func (vm *VM) GetAllSlots(obj *Object) Slots {
	slots := Slots{}
	obj.slots.foreach(vm, func(key string, value *syncSlot) bool {
		if value.value != nil {
			slots[key] = value.value
		}
		return true
	})
	return slots
}

// SetSlot sets the value of a slot on obj.
func (vm *VM) SetSlot(obj *Object, slot string, value *Object) {
	sy := obj.slots.open(vm, slot)
	sy.value = value
	sy.release()
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
	sy := obj.slots.open(vm, slot)
	return func(value *Object) {
		sy.value = value
		sy.release()
	}
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
	// TODO: create the trie directly instead of just setting each slot
	for slot, value := range slots {
		vm.SetSlot(obj, slot, value)
	}
}

// RemoveSlot removes slots from obj's local slots, if they are present.
func (vm *VM) RemoveSlot(obj *Object, slots ...string) {
	for _, slot := range slots {
		// TODO: only remove slots that exist
		vm.SetSlot(obj, slot, nil)
	}
}
