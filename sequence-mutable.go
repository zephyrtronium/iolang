package iolang

import (
	"fmt"
)

func (s *Sequence) CheckMutable(name string) error {
	if s.IsMutable() {
		return nil
	}
	return fmt.Errorf("'%s' cannot be called on an immutable sequence", name)
}

func (vm *VM) initMutableSeq(slots Slots) {
}