package gcc

import (
	"github.com/wnxd/microdbg-linux/gcc/arm64"
	"github.com/wnxd/microdbg-linux/gcc/x86_64"
	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
)

func NewVaList(dbg debugger.Debugger, ptr uint64) (debugger.Args, error) {
	switch dbg.Emulator().Arch() {
	case emulator.ARCH_ARM, emulator.ARCH_X86:
		return NewStdVaList(dbg, ptr, 4)
	case emulator.ARCH_ARM64:
		return arm64.NewVaList(dbg, ptr)
	case emulator.ARCH_X86_64:
		return x86_64.NewVaList(dbg, ptr)
	}
	return nil, emulator.ErrArchUnsupported
}
