package kernel_lp64

import (
	"sync"
	"unsafe"

	linux "github.com/wnxd/microdbg-linux"
	"github.com/wnxd/microdbg/emulator"
)

type sigset_t uint64

type sigaction struct {
	sa_handler  emulator.Uintptr64
	sa_flags    int32
	sa_restorer emulator.Uintptr64
	sa_mask     sigset_t
}

type signal struct {
	set   sigset_t
	rw    sync.RWMutex
	table map[int32]*sigaction
}

type siginfo_t struct {
	si_signo int32
	si_errno int32
	si_code  int32
	_si_pad  [29]int32
}

func (set *sigset_t) sigemptyset() {
	*set = 0
}

func (set *sigset_t) sigfillset() {
	*set = ^sigset_t(0)
}

func (set *sigset_t) sigaddset(sig int32) {
	*set |= 1 << uint(sig-1)
}

func (set *sigset_t) sigdelset(sig int32) {
	*set &^= 1 << uint(sig-1)
}

func (s *signal) ctor() {
	s.table = make(map[int32]*sigaction)
}

func (s *signal) dtor() {
	s.table = nil
}

func (s *signal) rt_sigaction(ctx linux.Context, signal int32, act, oldact emulator.Pointer, size uint64) int32 {
	action := new(sigaction)
	err := act.MemReadPtr(uint64(unsafe.Sizeof(*action)), unsafe.Pointer(action))
	if err != nil {
		ctx.SetErrno(linux.EFAULT)
		return -1
	}
	s.rw.Lock()
	old, ok := s.table[signal]
	s.table[signal] = action
	s.rw.Unlock()
	if oldact.IsNil() || !ok {
		return 0
	}
	oldact.MemWritePtr(uint64(unsafe.Sizeof(*old)), unsafe.Pointer(old))
	return 0
}

func (s *signal) rt_sigprocmask(ctx linux.Context, how int32, set, oldset emulator.Pointer, size uint64) int32 {
	const (
		SIG_BLOCK = iota + 1
		SIG_UNBLOCK
		SIG_SETMASK
	)

	if !oldset.IsNil() {
		err := oldset.MemWritePtr(uint64(unsafe.Sizeof(s.set)), unsafe.Pointer(&s.set))
		if err != nil {
			ctx.SetErrno(linux.EFAULT)
			return -1
		}
	}
	var value sigset_t
	err := set.MemReadPtr(uint64(unsafe.Sizeof(value)), unsafe.Pointer(&value))
	if err != nil {
		ctx.SetErrno(linux.EFAULT)
		return -1
	}
	switch how {
	case SIG_BLOCK:
		s.set |= value
	case SIG_UNBLOCK:
		s.set &^= value
	case SIG_SETMASK:
		s.set = value
	default:
		ctx.SetErrno(linux.EINVAL)
		return -1
	}
	return 0
}

func (s *signal) rt_tgsigqueueinfo(ctx linux.Context, tgid, tid, sig int32, info emulator.Pointer) int32 {
	var si siginfo_t
	print(unsafe.Sizeof(si))
	err := info.MemReadPtr(uint64(unsafe.Sizeof(si)), unsafe.Pointer(&si))
	if err != nil {
		ctx.SetErrno(linux.EFAULT)
		return -1
	}
	panic("rt_tgsigqueueinfo")
	ctx.SetErrno(linux.ENOSYS)
	return -1
}
