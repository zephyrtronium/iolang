package iolang

// SequenceSize is a Sequence method.
//
// size returns the number of items in the sequence.
func SequenceSize(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if s.IsMutable() {
		defer MutableMethod(target)()
	}
	return vm.NewNumber(float64(s.Len()))
}
