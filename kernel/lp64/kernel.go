package kernel_lp64

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

func (k *Kernel) KernelInit(dbg debugger.Debugger) error {
	var handleInterrupt debugger.InterruptCallback
	switch dbg.Emulator().Arch() {
	case emulator.ARCH_ARM64:
		handleInterrupt = k.handleArm64Interrupt
	default:
		return errors.ErrUnsupported
	}
	hook, err := dbg.AddHook(emulator.HOOK_TYPE_INTR, handleInterrupt, nil, 1, 0)
	if err != nil {
		return err
	}
	k.sys.ctor()
	k.intrHook = hook
	return nil
}

func (k *Kernel) KernelClose() error {
	k.intrHook.Close()
	k.sys.dtor()
	return nil
}

func (k *Kernel) handleArm64Interrupt(ctx debugger.Context, intno uint64, data any) debugger.HookResult {
	switch intno {
	case emu_arm.ARM_INTR_EXCP_SWI:
		return k.handleSwi(ctx)
	default:
		return debugger.HookResult_Next
	}
}

func (k *Kernel) handleSwi(ctx debugger.Context) debugger.HookResult {
	pc, err := ctx.RegRead(emu_arm64.ARM64_REG_PC)
	if err != nil {
		return debugger.HookResult_Next
	}
	var code uint32
	err = ctx.ToPointer(pc-4).MemReadPtr(4, unsafe.Pointer(&code))
	if err != nil {
		return debugger.HookResult_Next
	}
	swi := (code >> 5) & 0xffff
	switch swi {
	case 0:
		return k.syscall(ctx)
	}
	return debugger.HookResult_Next
}

func (k *Kernel) syscall(ctx debugger.Context) debugger.HookResult {
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
