package kernel

import (
	"errors"
	"unsafe"

	linux "github.com/wnxd/microdbg-linux"
	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
	emu_arm "github.com/wnxd/microdbg/emulator/arm"
	emu_arm64 "github.com/wnxd/microdbg/emulator/arm64"
)

type Kernel struct {
	sys      Syscall
	err      linux.Errno
	intrHook debugger.HookHandler
}

func NewKernel(dbg debugger.Debugger) (*Kernel, error) {
	k := new(Kernel)
	var handleIntr debugger.InterruptCallback
	switch dbg.Arch() {
	case emulator.ARCH_ARM:
		handleIntr = k.armIntr
	case emulator.ARCH_ARM64:
		handleIntr = k.arm64Intr
	default:
		return nil, errors.ErrUnsupported
	}
	hook, err := dbg.AddHook(emulator.HOOK_TYPE_INTR, handleIntr, nil, 1, 0)
	if err != nil {
		return nil, err
	}
	k.sys.ctor()
	k.intrHook = hook
	return k, nil
}

func (k *Kernel) Close() error {
	k.intrHook.Close()
	return k.sys.Close()
}

func (k *Kernel) NR(no uint64) linux.NR {
	return linux.NR(no)
}

func (k *Kernel) Syscall() linux.Syscall {
	return &k.sys
}

func (k *Kernel) Errno() linux.Errno {
	return k.err
}

func (k *Kernel) SetErrno(err linux.Errno) {
	k.err = err
}

func (k *Kernel) armIntr(ctx debugger.Context, intno uint64, data any) debugger.HookResult {
	const CPSR_T = 1 << 5

	if intno != emu_arm.ARM_INTR_EXCP_SWI {
		return debugger.HookResult_Next
	}
	pc_cpsr, err := ctx.RegReadBatch(emu_arm.ARM_REG_PC, emu_arm.ARM_REG_CPSR)
	if err != nil {
		return debugger.HookResult_Next
	}
	if pc_cpsr[1]&CPSR_T != 0 {
		var code uint16
		err = ctx.ToPointer(pc_cpsr[0]-2).MemReadPtr(2, unsafe.Pointer(&code))
		if err != nil {
			return debugger.HookResult_Next
		} else if swi := code & 0xff; swi != 0 {
			return debugger.HookResult_Next
		}
	} else {
		var code uint32
		err = ctx.ToPointer(pc_cpsr[0]-4).MemReadPtr(4, unsafe.Pointer(&code))
		if err != nil {
			return debugger.HookResult_Next
		} else if swi := code & 0xffffff; swi != 0 {
			return debugger.HookResult_Next
		}
	}
	nr, err := ctx.RegRead(emu_arm.ARM_REG_R7)
	if err != nil {
		return debugger.HookResult_Next
	}
	args, err := ctx.RegReadBatch(emu_arm.ARM_REG_R0, emu_arm.ARM_REG_R1, emu_arm.ARM_REG_R2, emu_arm.ARM_REG_R3, emu_arm.ARM_REG_R4, emu_arm.ARM_REG_R5)
	if err != nil {
		return debugger.HookResult_Next
	}
	dbg := ctx.Debugger().(linux.Debugger)
	call := dbg.Syscall().Get(dbg.NR(nr))
	if call == nil {
		return debugger.HookResult_Next
	}
	dbg.SetErrno(0)
	r := call(linux.NewContext(ctx, dbg), args...)
	ctx.RegWrite(emu_arm.ARM_REG_R0, r)
	return debugger.HookResult_Done
}

func (k *Kernel) arm64Intr(ctx debugger.Context, intno uint64, data any) debugger.HookResult {
	if intno != emu_arm.ARM_INTR_EXCP_SWI {
		return debugger.HookResult_Next
	}
	pc, err := ctx.RegRead(emu_arm64.ARM64_REG_PC)
	if err != nil {
		return debugger.HookResult_Next
	}
	var code uint32
	err = ctx.ToPointer(pc-4).MemReadPtr(4, unsafe.Pointer(&code))
	if err != nil {
		return debugger.HookResult_Next
	}
	if swi := (code >> 5) & 0xffff; swi != 0 {
		return debugger.HookResult_Next
	}
	nr, err := ctx.RegRead(emu_arm64.ARM64_REG_X8)
	if err != nil {
		return debugger.HookResult_Next
	}
	args, err := ctx.RegReadBatch(emu_arm64.ARM64_REG_X0, emu_arm64.ARM64_REG_X1, emu_arm64.ARM64_REG_X2, emu_arm64.ARM64_REG_X3, emu_arm64.ARM64_REG_X4, emu_arm64.ARM64_REG_X5)
	if err != nil {
		return debugger.HookResult_Next
	}
	dbg := ctx.Debugger().(linux.Debugger)
	call := dbg.Syscall().Get(dbg.NR(nr))
	if call == nil {
		return debugger.HookResult_Next
	}
	dbg.SetErrno(0)
	r := call(linux.NewContext(ctx, dbg), args...)
	ctx.RegWrite(emu_arm64.ARM64_REG_X0, r)
	return debugger.HookResult_Done
}
