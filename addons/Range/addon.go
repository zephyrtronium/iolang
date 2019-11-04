package Range

// Code generated by mkaddon; DO NOT EDIT

import (
	"bytes"
	"compress/zlib"

	"github.com/zephyrtronium/iolang"
)

// IoAddon returns a loader for the Range addon.
func IoAddon() iolang.Addon {
	return addonRange{}
}

type addonRange struct{}

func (addonRange) Name() string {
	return "Range"
}

var addonRangeProtos = []string{
	"Range",
}

func (addonRange) Protos() []string {
	return addonRangeProtos
}

var addonRangeDepends []string

func (addonRange) Depends() []string {
	return addonRangeDepends
}

func (addonRange) Init(vm *iolang.VM) {
	slotsRange := iolang.Slots{
		"at":       vm.NewCFunction(At, nil),
		"contains": vm.NewCFunction(Contains, nil),
		"first":    vm.NewCFunction(First, nil),
		"foreach":  vm.NewCFunction(Foreach, nil),
		"index":    vm.NewCFunction(Index, nil),
		"indexOf":  vm.NewCFunction(IndexOf, nil),
		"last":     vm.NewCFunction(Last, nil),
		"next":     vm.NewCFunction(Next, nil),
		"previous": vm.NewCFunction(Previous, nil),
		"rewind":   vm.NewCFunction(Rewind, nil),
		"setIndex": vm.NewCFunction(SetIndex, nil),
		"setRange": vm.NewCFunction(SetRange, nil),
		"size":     vm.NewCFunction(Size, nil),
		"value":    vm.NewCFunction(Value, nil),

		"type": vm.NewString("Range"),
	}
	vm.Install("Range", &iolang.Object{
		Slots:  slotsRange,
		Protos: []*iolang.Object{vm.BaseObject},
		Value:  Range{},
		Tag:    RangeTag,
	})

	for i, b := range addonRangeIo {
		r, err := zlib.NewReader(bytes.NewReader(b))
		if err != nil {
			panic(err)
		}
		exc, stop := vm.DoReader(r, addonRangeFiles[i])
		if stop == iolang.ExceptionStop {
			panic(exc)
		}
	}
}

var addonRangeFiles = []string{"io/range.io"}

var addonRangeIo = [][]byte{
	{0x78, 0x9c, 0x8c, 0x55, 0xcd, 0x6e, 0xe3, 0x38, 0xc, 0x3e, 0x6f, 0x9e, 0x82, 0xc8, 0x49, 0x42, 0xd5, 0x22, 0xb9, 0xee, 0xd6, 0x7b, 0x68, 0xb1, 0x87, 0x2, 0xdb, 0x4e, 0x31, 0xe9, 0xb, 0xa8, 0x36, 0x9d, 0xa8, 0x51, 0x64, 0x57, 0xa2, 0x8d, 0x64, 0x9e, 0x7e, 0x20, 0xca, 0x76, 0xec, 0x24, 0xed, 0xc4, 0x87, 0x38, 0x22, 0xe9, 0x8f, 0xe4, 0xc7, 0x1f, 0xfd, 0xd4, 0x6e, 0x8d, 0x50, 0x54, 0x62, 0x6, 0x0, 0xa0, 0xc3, 0xff, 0x26, 0x10, 0xfc, 0x9d, 0xc1, 0xe, 0x69, 0x53, 0x15, 0x49, 0x1a, 0x1f, 0x1b, 0x85, 0xd6, 0x4, 0x12, 0x72, 0x90, 0x5, 0xb4, 0x25, 0x94, 0x95, 0x47, 0x9d, 0x6f, 0x44, 0xab, 0xc0, 0x82, 0xae, 0x6b, 0x74, 0x85, 0x68, 0x65, 0x32, 0x92, 0x33, 0x7e, 0xed, 0x74, 0x3d, 0x82, 0xcc, 0xb5, 0xb5, 0x50, 0xa0, 0xc5, 0xb5, 0x26, 0x7c, 0xab, 0x9e, 0x93, 0x94, 0xb1, 0x92, 0x7b, 0x5, 0xf3, 0x9d, 0xae, 0x9f, 0xdc, 0xab, 0xd5, 0x39, 0xce, 0x65, 0x7, 0x12, 0xd0, 0x62, 0xce, 0xa1, 0x71, 0x88, 0x6b, 0xa4, 0x95, 0xad, 0x48, 0xcc, 0x93, 0x7c, 0xde, 0x5b, 0x59, 0x93, 0xe3, 0xc8, 0x59, 0x20, 0xed, 0x49, 0x41, 0xa0, 0xaa, 0x8e, 0xbf, 0x58, 0xab, 0xef, 0x33, 0x22, 0xac, 0x21, 0x4b, 0x2f, 0x53, 0xbe, 0x18, 0xfb, 0x5f, 0xab, 0xad, 0x58, 0x1e, 0xd, 0xca, 0xca, 0xb, 0x13, 0x91, 0x4e, 0x61, 0x8f, 0xc9, 0xa7, 0x4c, 0x48, 0x18, 0x39, 0xb0, 0x20, 0x67, 0xb3, 0xc7, 0xca, 0x23, 0xbc, 0x34, 0xbb, 0x77, 0xf4, 0xcc, 0xf6, 0x5f, 0x54, 0x8d, 0xc2, 0x44, 0x57, 0xa8, 0x44, 0x27, 0x55, 0xf, 0x87, 0x74, 0x5c, 0x4a, 0x19, 0xad, 0x1e, 0xe, 0x67, 0x76, 0xec, 0x2f, 0x15, 0x2e, 0xb7, 0x95, 0x43, 0x8, 0x48, 0x7c, 0x64, 0xdf, 0xa, 0x6, 0xab, 0x9e, 0x3b, 0x87, 0x7b, 0x7a, 0x72, 0x2b, 0xfc, 0x6c, 0xd0, 0x4d, 0xe9, 0xd9, 0x9a, 0x11, 0x23, 0xf1, 0x94, 0xd2, 0x66, 0x5, 0x64, 0xb0, 0x3c, 0x29, 0xf6, 0xd, 0xdb, 0x9c, 0x64, 0x35, 0xe0, 0xf6, 0x5d, 0x74, 0xa5, 0x3b, 0x53, 0x26, 0xae, 0x82, 0xf9, 0x85, 0x90, 0x65, 0xb0, 0x50, 0xe0, 0x91, 0x1a, 0xef, 0x92, 0x2f, 0x4e, 0x4d, 0x5e, 0x17, 0x9c, 0x45, 0x5d, 0x18, 0xb7, 0x8e, 0xae, 0x16, 0x97, 0xdb, 0x33, 0x3f, 0xfa, 0x8d, 0xcf, 0x46, 0xb7, 0x1c, 0x58, 0xa9, 0x6d, 0xc0, 0x89, 0x26, 0x74, 0x61, 0xaf, 0x90, 0xc2, 0xf0, 0x75, 0xc0, 0x58, 0x6d, 0x24, 0xc8, 0x2b, 0x47, 0xda, 0xb8, 0x20, 0x72, 0x9, 0xa6, 0x7c, 0xf3, 0xd, 0xa, 0x86, 0xca, 0x80, 0x7c, 0x83, 0xff, 0xc0, 0xbb, 0x47, 0xbd, 0x95, 0x72, 0x82, 0x68, 0x4a, 0xb6, 0x51, 0x49, 0xa9, 0x86, 0x60, 0xb3, 0xe1, 0xdf, 0xcd, 0x38, 0x17, 0x39, 0x26, 0x68, 0xb0, 0xcd, 0x60, 0xe0, 0xea, 0x7b, 0x9a, 0xc8, 0xc7, 0xbc, 0x58, 0x85, 0xfb, 0x55, 0x1c, 0x87, 0x1e, 0x44, 0x82, 0xe, 0xcf, 0xd, 0xe9, 0x77, 0x7b, 0xcc, 0x58, 0x44, 0x7b, 0x2e, 0xc0, 0x2d, 0x2c, 0x65, 0xea, 0xbe, 0x85, 0x82, 0xdb, 0xa5, 0x1c, 0x72, 0xdf, 0x4e, 0x99, 0x2b, 0x62, 0xc3, 0x5d, 0xcf, 0x9c, 0xd3, 0x3b, 0x64, 0xea, 0xa6, 0x28, 0xf1, 0xd9, 0xa7, 0x40, 0x9, 0x8c, 0x2b, 0x70, 0xff, 0xa3, 0xe4, 0x58, 0x34, 0x89, 0xad, 0x94, 0x5d, 0xa1, 0x23, 0xdb, 0xc6, 0x35, 0x28, 0x23, 0x43, 0x67, 0xdf, 0x9b, 0x52, 0xec, 0xe1, 0x9e, 0x11, 0x98, 0x96, 0x33, 0x83, 0x9e, 0xf, 0x4d, 0xaf, 0xd, 0x89, 0x6d, 0xaa, 0xa0, 0x26, 0xb1, 0x3f, 0x29, 0xd0, 0x24, 0xb5, 0x54, 0xc9, 0x33, 0xfd, 0xf5, 0xe8, 0x8b, 0x2f, 0xd0, 0x4d, 0x29, 0xb6, 0xa9, 0xcd, 0x2f, 0xaa, 0x7b, 0xb8, 0xda, 0x63, 0x5c, 0x23, 0x2b, 0xfc, 0x14, 0x1d, 0xa0, 0x29, 0x99, 0xc5, 0xf8, 0xed, 0xbc, 0x30, 0x6b, 0x43, 0xfd, 0x64, 0xcd, 0x15, 0x2c, 0x15, 0x2c, 0x64, 0x2c, 0xec, 0xe3, 0x46, 0x7b, 0x9d, 0x13, 0xfa, 0xcb, 0xce, 0xff, 0x94, 0x1e, 0x4c, 0x1a, 0xef, 0x6b, 0x9, 0xb7, 0xf0, 0xec, 0x6b, 0x1b, 0xf6, 0xd1, 0xd, 0x46, 0x9a, 0x85, 0xb, 0x7d, 0x3d, 0x69, 0xce, 0xc5, 0x30, 0x10, 0x12, 0xee, 0xee, 0x78, 0x29, 0xc4, 0xf1, 0xfe, 0x37, 0xa6, 0x16, 0xf9, 0x98, 0x2e, 0x93, 0xa4, 0x8c, 0xbd, 0xca, 0xda, 0xc9, 0x35, 0x63, 0xb1, 0x45, 0x17, 0x36, 0x84, 0xc6, 0x5d, 0xba, 0xc1, 0x4c, 0x29, 0x2a, 0xda, 0x60, 0xd7, 0xee, 0xf7, 0x17, 0xe6, 0x29, 0xa9, 0x47, 0x30, 0xbc, 0x9f, 0x46, 0xe5, 0x6c, 0x79, 0xc1, 0x0, 0x55, 0xc7, 0xc5, 0x25, 0xbb, 0x4b, 0x6b, 0xb0, 0x49, 0x20, 0x7d, 0xfb, 0x1b, 0x5, 0x87, 0x69, 0xbd, 0x9b, 0x88, 0xd1, 0x4e, 0x44, 0x2d, 0x74, 0x17, 0x91, 0x99, 0x6e, 0x2, 0x38, 0x5d, 0x62, 0x1f, 0xa, 0xf6, 0xe7, 0xdd, 0xd3, 0xf6, 0x37, 0x8f, 0x68, 0x63, 0xbb, 0x7c, 0xf0, 0xb4, 0x48, 0xd8, 0x19, 0x27, 0x1a, 0x16, 0xa4, 0xf3, 0x89, 0x30, 0xa, 0x78, 0x82, 0xb2, 0xc, 0xe, 0x2a, 0x2e, 0xdf, 0xa5, 0x3c, 0xe9, 0xdc, 0x4b, 0xb5, 0x6b, 0xc1, 0xea, 0x2e, 0xdb, 0x78, 0x3, 0xfc, 0xe, 0x0, 0x0, 0xff, 0xff, 0x38, 0xef, 0x29, 0x96},
}
