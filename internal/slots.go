package internal

/*
This file contains the implementation of slot lookups. Executing Io code
amounts to looking up a slot and then calling a function pointer, and it turns
out that the latter is cheap. Slots are the expensive part of the hot path, so
they are the primary target for optimization.

If you are here to read how this works, good luck and have fun; I tried hard to
document everything thoroughly. If you're trying to make changes, have some
results from my debugging experiences:

panic: iolang: error executing initialization code from io/98_Debugger.io: Coroutine does not respond to setRunTarget (exception)
	This happens when getSlot("self") returns nil, which probably means that
Locals slots aren't set up correctly, which probably means that the underlying
implementation of foreachSlot is broken.
	This failure might be intermittent (but still happening much more often
than not), depending on the exact nature of the problem with foreachSlot,
because the order in which slots are created for Core protos is generally
random. If getSlot is among the first few slots created, it is much more likely
for a broken foreachSlot to still copy it to Locals, so the getSlot message
doesn't need to be forwarded and thus will look on the locals instead of on
self.
	The reason this happens in 98_Debugger.io is because that is the first
place where initialization code ends up using a setter created by newSlot.

panic: iolang: no Core proto named CFunction
	The first call to vm.NewCFunction involves the first slot lookup during VM
initialization, via vm.CoreProto, which panics with this message if
GetLocalSlot fails to find the slot. This means that at least one of
vm.localSyncSlot and vm.SetSlot is broken.
*/

import (
	"math/bits"
	"sync"
	"sync/atomic"
	"unsafe"
)

// protoLink is a node of a concurrent linked list. Its fields
// must be accessed atomically.
type protoLink struct {
	// p is the proto at this element.
	p *Object

	// n is the link to the next node.
	n *protoLink
	// mu is a mutex for write permission on the link (but not the node, which
	// should always be written atomically). This prevents problems relating to
	// concurrent insertions and deletions on the list. Even while holding the
	// lock, the node's data and link fields must be handled atomically, as
	// readers do not acquire the lock.
	mu sync.Mutex
}

// logicalDeleted is a special value used to mark the head of a list as
// logically deleted. Readers should spin while they see that the head proto is
// equal to this value, and therefore writers should avoid using it.
var logicalDeleted = new(Object)

// protoHead returns the object's first proto and the link to the next.
func (o *Object) protoHead() (p *Object, n *protoLink) {
	for {
		p = (*Object)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p))))
		if p == logicalDeleted {
			// If the head is logically deleted, then someone holds its lock.
			// Rather than simply spinning while waiting for it to become valid
			// again, we can try to acquire the lock so the runtime can park
			// this goroutine. Then, since we hold the lock, we can be certain
			// the node is in a valid state and there are no concurrent
			// writers, so we can read it non-atomically.
			o.protos.mu.Lock()
			p = o.protos.p
			o.protos.mu.Unlock()
		}
		n = (*protoLink)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.n))))
		// In order to ensure consistency, we require the current proto to
		// match the one we got at the start, otherwise we might return the
		// first proto corresponding to one list and the link of another.
		if p == (*Object)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p)))) {
			return p, n
		}
		// If we don't have such a match, then it's fine to just try again,
		// because protos are rarely modified.
	}
}

// iterR returns the node's data and its next link. p is nil when the iteration
// reaches the end of the list. This method is suitable only when not modifying
// the list, as it may follow links that are being modified. This method must
// not be called on the list head rooted on an object; use protoHead instead.
func (l *protoLink) iterR() (p *Object, n *protoLink) {
	if l == nil {
		return nil, nil
	}
	p = (*Object)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&l.p))))
	n = (*protoLink)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&l.n))))
	return p, n
}

// Protos returns a snapshot of the object's protos. The result is nil if o has
// no protos. This is more efficient than using ForeachProto to construct a
// slice.
func (o *Object) Protos() []*Object {
	p, n := o.protoHead()
	if p == nil {
		return nil
	}
	r := []*Object{p} // vast majority of objects have one proto
	for p, n = n.iterR(); p != nil; p, n = n.iterR() {
		r = append(r, p)
	}
	return r
}

// ForeachProto calls exec on each of the object's protos. exec must not modify
// o's protos list. If exec returns false, then the iteration ceases.
func (o *Object) ForeachProto(exec func(p *Object) bool) {
	p, n := o.protoHead()
	if p == nil || !exec(p) {
		return
	}
	for p, n := n.iterR(); p != nil; p, n = n.iterR() {
		if !exec(p) {
			return
		}
	}
}

// SetProtos sets the object's protos to those given.
func (o *Object) SetProtos(protos ...*Object) {
	o.protos.mu.Lock()
	switch len(protos) {
	case 0:
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p)), nil)
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.n)), nil)
	case 1:
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p)), unsafe.Pointer(logicalDeleted))
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.n)), nil)
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p)), unsafe.Pointer(protos[0]))
	default:
		n := &protoLink{p: protos[1]}
		m := n
		for i := 2; i < len(protos); i++ {
			n.n = &protoLink{p: protos[i]}
			n = n.n
		}
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p)), unsafe.Pointer(logicalDeleted))
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.n)), unsafe.Pointer(m))
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p)), unsafe.Pointer(protos[0]))
	}
	o.protos.mu.Unlock()
}

// AppendProto appends a proto to the end of the object's protos list.
func (o *Object) AppendProto(proto *Object) {
	o.protos.mu.Lock()
	// Try swapping in a new head if there isn't one.
	if atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p)), nil, unsafe.Pointer(proto)) {
		o.protos.mu.Unlock()
		return
	}
	// Since we acquired the head's lock, the head cannot be logically deleted,
	// and since we acquire each successive node's lock, there are no
	// concurrent writers, so we can load links without atomics.
	cur := &o.protos
	next := cur.n
	for next != nil {
		next.mu.Lock()
		cur.mu.Unlock()
		cur = next
		next = cur.n
	}
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&cur.n)), unsafe.Pointer(&protoLink{p: proto}))
	cur.mu.Unlock()
}

// PrependProto prepends a proto to the front of the object's protos list.
func (o *Object) PrependProto(proto *Object) {
	o.protos.mu.Lock()
	old := (*Object)(atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p)), unsafe.Pointer(logicalDeleted)))
	if old == nil {
		// There was no head.
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p)), unsafe.Pointer(proto))
		o.protos.mu.Unlock()
		return
	}
	next := &protoLink{p: old, n: o.protos.n}
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.n)), unsafe.Pointer(next))
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p)), unsafe.Pointer(proto))
	o.protos.mu.Unlock()
}

// RemoveProto removes all instances of a proto from the object's protos list.
// Comparison is done by identity only.
func (o *Object) RemoveProto(proto *Object) {
	// This could be more optimized, but it's a bit too subtle to ensure
	// atomicity – and too rare of a call – to justify the effort.
	o.protos.mu.Lock()
	var r []*Object
	// Note that we can call ForeachProto while the head's mutex is held
	// because ForeachProto only locks via protoHead, which in turn only locks
	// when o.protos.p is logicalDeleted, which only happens while there's
	// another active writer, which requires the lock itself.
	o.ForeachProto(func(p *Object) bool {
		if p != proto {
			r = append(r, p)
		}
		return true
	})
	// However, we can't call SetProtos while the mutex is held.
	switch len(r) {
	case 0:
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p)), nil)
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.n)), nil)
	case 1:
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p)), unsafe.Pointer(logicalDeleted))
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.n)), nil)
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p)), unsafe.Pointer(r[0]))
	default:
		n := &protoLink{p: r[1]}
		m := n
		for i := 2; i < len(r); i++ {
			n.n = &protoLink{p: r[i]}
			n = n.n
		}
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p)), unsafe.Pointer(logicalDeleted))
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.n)), unsafe.Pointer(m))
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.protos.p)), unsafe.Pointer(r[0]))
	}
	o.protos.mu.Unlock()
}

// syncSlot is a synchronized slot. Once a particular VM accesses this slot,
// it becomes the slot's owner for the duration of the access, and other coros
// must wait until the owner releases it before they can access the slot.
//
// To set the slot value, the current coroutine must claim the slot and then
// set the value atomically. To read the slot value, the current coroutine must
// claim the slot and then may read the value non-atomically. To check whether
// the slot is valid, any coroutine may atomically load its value and check
// whether it is nil without claiming, but it must claim the slot to use its
// value regardless, hence a second validity check must follow claiming.
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

// snap returns a snapshot of the value in s. This claims the slot, retrieves
// its value, and then releases the slot.
func (s *syncSlot) snap(vm *VM) *Object {
	s.claim(vm)
	r := s.load()
	s.release()
	return r
}

// load returns the slot's value. The slot must be claimed by the current coro.
func (s *syncSlot) load() *Object {
	return s.value
}

// set sets the slot's value. The slot must be claimed by the current coro.
func (s *syncSlot) set(v *Object) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&s.value)), unsafe.Pointer(v))
}

// valid returns true if the slot currently holds a value.
func (s *syncSlot) valid() bool {
	return atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.value))) != nil
}

// SyncSlot proxies synchronized access to a single slot. A SyncSlot value may
// be copied, but it must never be accessed with a VM other than the one which
// originally obtained it.
type SyncSlot struct {
	sy    *syncSlot
	owner *VM
}

// Lock locks the slot for the owning coroutine. The same coroutine may lock
// the slot multiple times without blocking, but each call to Lock must be
// followed eventually by exactly one corresponding call to Unlock. Each Sync
// slot accessor logically makes one call to Lock.
func (s SyncSlot) Lock() {
	s.sy.claim(s.owner)
}

// Unlock unlocks the slot. Each call to Unlock must be preceded by exactly one
// call to Lock or any of the Sync slot accessors. In the latter case, Unlock
// should be called even if the slot is not valid.
func (s SyncSlot) Unlock() {
	if s.sy != nil {
		s.sy.release()
	}
}

// Load obtains the slot's value. The slot must be locked.
func (s SyncSlot) Load() *Object {
	return s.sy.load()
}

// Snap returns a snapshot of the slot's value by locking, loading, and
// unlocking it. If the slot is not valid, the result is nil, and Snap may or
// may not block.
func (s SyncSlot) Snap() *Object {
	if s.sy == nil {
		return nil
	}
	return s.sy.snap(s.sy.owner)
}

// Set sets the slot's value. The slot must be locked. Setting a value of nil
// marks the slot as deleted.
func (s SyncSlot) Set(value *Object) {
	s.sy.set(value)
}

// Delete calls Set(nil). The slot must be locked.
func (s SyncSlot) Delete() {
	s.Set(nil)
}

// Valid returns true if the slot currently has a value or false if it is
// deleted. The slot does not need to be locked, but its validity may change at
// any time if it is not locked.
func (s SyncSlot) Valid() bool {
	return s.sy != nil && s.sy.valid()
}

// actualSlots is a synchronized trie structure that implements slots.
//
// The trie is grow-only. A leaf value of nil indicates an empty slot that has
// not been created; a slot value of nil indicates an unset or deleted slot.
type actualSlots struct {
	root *slotBranch // atomic
}

// slotBranch holds a level of the slots trie.
type slotBranch struct {
	// leaf is the value associated with the string ending at the current node.
	// Once this branch is connected into the trie, leaf must be accessed
	// atomically.
	leaf *syncSlot

	// scut is a read-only shortcut to the leaf that justified creating this
	// branch, if there is one.
	scut *syncSlot
	// scutName is the name of the shortcut slot beginning at this node's
	// parent edge value.
	scutName string

	// rec is the first slotRecord. It is included as a value rather than as a
	// pointer to save an indirection on each lookup.
	rec slotRecord

	// hasZero is an atomic flag indicating whether this branch has an edge
	// corresponding to the zero byte. If this is nonzero, then readers should
	// spin until zero is not nil.
	//
	// This is typed as a uintptr instead of a smaller type; no other type
	// would save space due to alignment requirements anyway.
	hasZero uintptr
	// zero is the zero edge.
	zero *slotBranch
}

// slotRecord represents one piece of the slots trie. Its fields must be
// manipulated atomically.
type slotRecord struct {
	// mask is the list of the names of this record's child nodes. A zero byte
	// indicates no edge.
	mask uintptr
	// children is this record's child nodes.
	children [recordChildren]*slotBranch
	// sibling is the next record at the same level in the trie.
	sibling *slotRecord
}

// recordChildren is the number of children in a single slotRecord, i.e. the
// size of uintptr in bytes.
const recordChildren = 4 << (^uintptr(0) >> 32 & 1)

// recordPop has the first bit in each byte set and all others clear.
const recordPop = ^uintptr(0) / 0xff

// load finds the given slot, or returns nil if there is no such slot. This may
// return a slot for which valid() returns false.
func (s *actualSlots) load(slot string) *syncSlot {
	branch := (*slotBranch)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.root))))
	if branch == nil {
		return nil
	}
	// Iterate manually rather than by range; we want bytes, not runes.
	for i := 0; i < len(slot); i++ {
		// Check the shortcut.
		if branch.scut != nil && branch.scutName == slot[i:] {
			return branch.scut
		}
		c := slot[i]
		if c == 0 {
			// The nul edge is special and lives on the branch itself.
			if atomic.LoadUintptr(&branch.hasZero) == 0 {
				return nil
			}
			next := (*slotBranch)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&branch.zero))))
			for next == nil {
				next = (*slotBranch)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&branch.zero))))
			}
			branch = next
			continue
		}
		cur := &branch.rec
		m := uintptr(c) * recordPop
		for {
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
				if cur == nil {
					return nil
				}
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
			next := (*slotBranch)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.children[k]))))
			for next == nil {
				// The edge to this child exists, but the node hasn't been set.
				// Another goroutine must be in the process of setting it. Spin
				// until it's available.
				next = (*slotBranch)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.children[k]))))
			}
			branch = next
			break
		}
	}
	return (*syncSlot)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&branch.leaf))))
}

// open loads the current value of the given slot if it exists or creates a new
// slot there if it does not. The slot is claimed by vm.
func (s *actualSlots) open(vm *VM, slot string) *syncSlot {
	branch := (*slotBranch)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.root))))
	if branch == nil {
		// Slots are empty. Try to create them, but it's possible we're not the
		// only ones doing so; whoever gets there first wins.
		branch = &slotBranch{}
		if !atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&s.root)), nil, unsafe.Pointer(branch)) {
			branch = (*slotBranch)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.root))))
		}
	}
	// Iterate manually rather than by range; we want bytes, not runes.
	for i := 0; i < len(slot); i++ {
		c := slot[i]
		if c == 0 {
			if !atomic.CompareAndSwapUintptr(&branch.hasZero, 0, 1) {
				next := (*slotBranch)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&branch.zero))))
				for next == nil {
					next = (*slotBranch)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&branch.zero))))
				}
				branch = next
				continue
			}
			node, leaf := recordBranch(vm, slot[i+1:])
			atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&branch.zero)), unsafe.Pointer(node))
			return leaf
		}
		cur := &branch.rec
		m := uintptr(c) * recordPop
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
					v = (cm - recordPop) &^ cm & (recordPop << 7)
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
			next := (*slotBranch)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.children[k]))))
			for next == nil {
				// The edge to this child exists, but the node hasn't been set.
				// Another goroutine must be in the process of setting it. Spin
				// until it's available.
				next = (*slotBranch)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.children[k]))))
			}
			branch = next
			break
		}
	}
	sy := (*syncSlot)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&branch.leaf))))
	if sy == nil {
		// The node existed, but its value was unset. This happens if the slot
		// we're creating is a prefix of a slot that was added earlier.
		sy = newSy(vm, nil)
		if atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&branch.leaf)), nil, unsafe.Pointer(sy)) {
			return sy
		}
		// Someone else created the slot before us. Use theirs.
		sy = (*syncSlot)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&branch.leaf))))
	}
	sy.claim(vm)
	return sy
}

// ForeachSlot executes a function for each slot on obj. If exec returns false,
// then the iteration ceases. The slots passed to exec are not locked and may
// not be valid, but it is guaranteed that calling Lock on them will not cause
// a nil dereference.
func (vm *VM) ForeachSlot(obj *Object, exec func(name string, sy SyncSlot) bool) {
	cur := (*slotBranch)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&obj.slots.root))))
	if cur == nil {
		return
	}
	cur.foreachIter(vm, exec, nil)
}

// foreachIter executes vm.ForeachSlot for a single depth of the trie.
func (r *slotBranch) foreachIter(vm *VM, exec func(name string, sy SyncSlot) bool, b []byte) []byte {
	sy := (*syncSlot)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&r.leaf))))
	if sy != nil && sy.valid() {
		if !exec(string(b), SyncSlot{sy: sy, owner: vm}) {
			return nil
		}
	}
	// Handle the zero edge specially.
	zero := (*slotBranch)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&r.zero))))
	if zero != nil {
		b = append(b, 0)
		b = zero.foreachIter(vm, exec, b)
		if b == nil {
			return nil
		}
		b = b[:len(b)-1]
	}
	cur := &r.rec
	for cur != nil {
		for k := 0; k < recordChildren; k++ {
			cm := atomic.LoadUintptr(&cur.mask)
			c := byte(cm >> (k * 8))
			if c == 0 {
				return b
			}
			r = (*slotBranch)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.children[k]))))
			if r == nil {
				continue
			}
			b = append(b, c)
			b = r.foreachIter(vm, exec, b)
			if b == nil {
				return nil
			}
			b = b[:len(b)-1]
		}
		cur = (*slotRecord)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cur.sibling))))
	}
	return b
}

// recordBranch allocates an entire new branch of a slots trie.
func recordBranch(vm *VM, branch string) (trunk *slotBranch, slot *syncSlot) {
	b := make([]slotBranch, len(branch)+1)
	trunk = &b[0]
	cur := trunk
	for i := 0; i < len(branch); i++ {
		cur.rec = slotRecord{
			mask:     uintptr(branch[i]),
			children: [recordChildren]*slotBranch{0: &b[i+1]},
		}
		cur = cur.rec.children[0]
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
// proto is nil if and only if the slot was not found.
func (vm *VM) GetSlot(obj *Object, slot string) (value, proto *Object) {
	if obj == nil {
		return nil, nil
	}
	// Check obj itself before using the graph traversal mechanisms.
	if sy := vm.localSyncSlot(obj, slot); sy != nil {
		value = sy.load()
		sy.release()
		return value, obj
	}
	sy, proto := vm.getSlotAncestor(obj, slot)
	if proto != nil {
		value = sy.load()
		sy.release()
	}
	return
}

// GetSlotSync is like GetSlot, but it returns a SyncSlot object rather than
// the slot value directly. The SyncSlot is locked at the time GetSlotSync
// returns. Other VMs will block on attempts to read or write the same slot,
// whether it is on obj or an ancestor, until the SyncSlot is unlocked. The
// slot is guaranteed to be valid if it is not nil. proto is nil if and only if
// the slot is not found.
func (vm *VM) GetSlotSync(obj *Object, slot string) (s SyncSlot, proto *Object) {
	if obj == nil {
		return
	}
	if sy := vm.localSyncSlot(obj, slot); sy != nil {
		return SyncSlot{sy: sy, owner: vm}, obj
	}
	sy, proto := vm.getSlotAncestor(obj, slot)
	if proto != nil {
		s = SyncSlot{sy: sy, owner: vm}
	}
	return
}

// getSlotAncestor finds a slot on obj's ancestors.
func (vm *VM) getSlotAncestor(obj *Object, slot string) (sy *syncSlot, proto *Object) {
	vm.protoSet.Reset()
	vm.protoSet.Add(obj.UniqueID())
	obj, link := obj.protoHead()
	if obj == nil {
		return nil, nil
	}
	// Append protos onto the stack in reverse order. To do this, we first
	// append them in forward order, then reverse the ones we added on.
	start := len(vm.protoStack)
	if vm.protoSet.Add(obj.UniqueID()) {
		vm.protoStack = append(vm.protoStack, obj)
	}
	for p, link := link.iterR(); p != nil; p, link = link.iterR() {
		if vm.protoSet.Add(p.UniqueID()) {
			vm.protoStack = append(vm.protoStack, p)
		}
	}
	n := (len(vm.protoStack) - start) / 2
	for i := 0; i < n; i++ {
		vm.protoStack[start+i], vm.protoStack[len(vm.protoStack)-i-1] = vm.protoStack[len(vm.protoStack)-i-1], vm.protoStack[start+i]
	}
	for len(vm.protoStack) > 0 {
		obj = vm.protoStack[len(vm.protoStack)-1] // grab the top
		if sy := vm.localSyncSlot(obj, slot); sy != nil {
			vm.protoStack = vm.protoStack[:0]
			return sy, obj
		}
		vm.protoStack = vm.protoStack[:len(vm.protoStack)-1] // actually pop
		start = len(vm.protoStack)
		p, link := obj.protoHead()
		if p == nil {
			continue
		}
		if vm.protoSet.Add(p.UniqueID()) {
			vm.protoStack = append(vm.protoStack, p)
		}
		for p, link := link.iterR(); link != nil; p, link = link.iterR() {
			if vm.protoSet.Add(p.UniqueID()) {
				vm.protoStack = append(vm.protoStack, p)
			}
		}
		n := (len(vm.protoStack) - start) / 2
		for i := 0; i < n; i++ {
			vm.protoStack[start+i], vm.protoStack[len(vm.protoStack)-i-1] = vm.protoStack[len(vm.protoStack)-i-1], vm.protoStack[start+i]
		}
	}
	return nil, nil
}

// GetLocalSlot checks only obj's own slots for a slot.
func (vm *VM) GetLocalSlot(obj *Object, slot string) (value *Object, ok bool) {
	if obj == nil {
		return nil, false
	}
	if sy := vm.localSyncSlot(obj, slot); sy != nil {
		value = sy.load()
		sy.release()
		return value, true
	}
	return nil, false
}

// GetLocalSlotSync is like GetLocalSlot, but it returns a SyncSlot object
// rather than the slot value directly. The SyncSlot is valid if and only if
// the slot exists on obj, and it is locked if it is valid. Other VMs will
// block on attempts to read or write the same slot until the SyncSlot is
// unlocked.
func (vm *VM) GetLocalSlotSync(obj *Object, slot string) SyncSlot {
	if obj == nil {
		return SyncSlot{}
	}
	sy := vm.localSyncSlot(obj, slot)
	if sy != nil {
		return SyncSlot{sy: sy, owner: vm}
	}
	return SyncSlot{}
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
	vm.ForeachSlot(obj, func(key string, value SyncSlot) bool {
		value.Lock()
		if value.Valid() {
			slots[key] = value.Load()
		}
		value.Unlock()
		return true
	})
	return slots
}

// SetSlot sets the value of a slot on obj.
func (vm *VM) SetSlot(obj *Object, slot string, value *Object) {
	sy := obj.slots.open(vm, slot)
	sy.set(value)
	sy.release()
}

// SetSlotSync returns a locked SyncSlot for the given slot on obj, creating
// the slot if it does not exist. The SyncSlot is not guaranteed to be valid,
// so its current value should not be accessed. Users should call SetSlotSync
// before evaluating Io messages that will determine the value, e.g.:
//
// 	sy := vm.SetSlotSync(obj, slot)
// 	defer sy.Unlock()
// 	value, stop := msg.Eval(vm, locals)
// 	if stop != NoStop {
// 		return vm.Stop(value, stop)
// 	}
// 	sy.Set(value)
func (vm *VM) SetSlotSync(obj *Object, slot string) SyncSlot {
	sy := obj.slots.open(vm, slot)
	return SyncSlot{sy: sy, owner: vm}
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
		sy := vm.GetLocalSlotSync(obj, slot)
		if sy.Valid() {
			sy.Delete()
		}
		sy.Unlock()
	}
}

// RemoveAllSlots removes all slots from obj in a single operation.
func (vm *VM) RemoveAllSlots(obj *Object) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&obj.slots.root)), nil)
}
