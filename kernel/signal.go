package kernel

import (
	"sync"

	linux "github.com/wnxd/microdbg-linux"
)

type sigset_t ulong_t

type sigaction struct {
	sa_handler  uintptr
	sa_flags    int32
	sa_mask     sigset_t
	sa_restorer uintptr
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

func (s *signal) rt_sigaction(ctx linux.Context, signal int32, act, oldact emuptr, size size_t) int32 {
	dbg := ctx.Debugger()
	action := new(sigaction)
	err := dbg.MemExtract(act, action)
	if err != nil {
		ctx.SetErrno(linux.EFAULT)
		return -1
	}
	s.rw.Lock()
	old, ok := s.table[signal]
	s.table[signal] = action
	s.rw.Unlock()
	if oldact == emunullptr || !ok {
		return 0
	}
	dbg.MemWrite(oldact, *old)
	return 0
}

func (s *signal) rt_sigprocmask(ctx linux.Context, how int32, set, oldset emuptr, size size_t) int32 {
	const (
		SIG_BLOCK = iota + 1
		SIG_UNBLOCK
		SIG_SETMASK
	)

	dbg := ctx.Debugger()
	if oldset != emunullptr {
		_, err := dbg.MemWrite(oldset, s.set)
		if err != nil {
			ctx.SetErrno(linux.EFAULT)
			return -1
		}
	}
	var value sigset_t
	err := dbg.MemExtract(set, &value)
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

func (s *signal) rt_tgsigqueueinfo(ctx linux.Context, tgid, tid, sig int32, info emuptr) int32 {
	var si siginfo_t
	dbg := ctx.Debugger()
	err := dbg.MemExtract(info, &si)
	if err != nil {
		ctx.SetErrno(linux.EFAULT)
		return -1
	}
	panic("rt_tgsigqueueinfo")
	ctx.SetErrno(linux.ENOSYS)
	return -1
}
