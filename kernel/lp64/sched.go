package kernel_lp64

import (
	"fmt"
	"sync"
	"unsafe"

	linux "github.com/wnxd/microdbg-linux"
	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
	emu_arm64 "github.com/wnxd/microdbg/emulator/arm64"
)

type sched struct {
	tasks sync.Map
}

func (s *sched) clone(ctx linux.Context, flags int32, child_stack, parent_tid, tls, child_tid emulator.Pointer) int32 {
	const (
		CLONE_VM     = 0x00000100
		CLONE_VFORK  = 0x00004000
		CLONE_SETTLS = 0x00080000
	)

	task, err := ctx.TaskFork()
	if err != nil {
		ctx.SetErrno(linux.EAGAIN)
		return -1
	}
	dbg := ctx.Debugger()
	taskCtx := task.Context()
	if child_stack.IsNil() {
		taskCtx.RetWrite(nil)
	} else {
		stack := child_stack.Address()
		taskCtx.RegWrite(taskCtx.SP(), stack)
		var call struct{ fn, arg uintptr }
		dbg.MemExtract(stack, &call)
		err = dbg.CallTaskOf(task, uint64(call.fn))
		if err != nil {
			task.Close()
			ctx.SetErrno(linux.EAGAIN)
			return -1
		}
		taskCtx.ArgWrite(debugger.Calling_Default, call.arg)
	}
	if flags&CLONE_SETTLS != 0 {
		switch dbg.Emulator().Arch() {
		case emulator.ARCH_ARM64:
			taskCtx.RegWrite(emu_arm64.ARM64_REG_TPIDR_EL0, tls.Address())
		case emulator.ARCH_X86_64:
		}
	}
	err = task.Run()
	if err != nil {
		task.Close()
		ctx.SetErrno(linux.EAGAIN)
		return -1
	}
	pid := int32(task.ID())
	if !child_tid.IsNil() {
		child_tid.MemWritePtr(4, unsafe.Pointer(&pid))
	}
	if flags&CLONE_VFORK != 0 {
		<-task.Done()
		fmt.Println(task.Err())
		task.Close()
	} else {
		s.tasks.Store(pid, task)
		go func() {
			<-task.Done()
			s.tasks.Delete(pid)
			fmt.Println(task.Err())
			task.Close()
		}()
	}
	return pid
}

func (s *sched) execve(ctx linux.Context, filename, argv, envp emulator.Pointer) int32 {
	ctx.SetErrno(linux.ENOSYS)
	return -1
}
