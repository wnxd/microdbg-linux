package internal

import (
	_ "unsafe"

	"github.com/wnxd/microdbg/emulator"
	"github.com/wnxd/microdbg/encoding"
)

//go:linkname PointerStream github.com/wnxd/microdbg/internal/debugger.PointerStream
func PointerStream(ptr emulator.Pointer, alloc func(uint64) (emulator.Pointer, error), size int) encoding.Stream
