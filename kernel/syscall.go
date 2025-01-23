package kernel

import (
	"math"

	linux "github.com/wnxd/microdbg-linux"
	"github.com/wnxd/microdbg/emulator"
)

type Syscall struct {
	fcntl
	futex
	signal
	prctl
	socket
	mman
	sched
}

func NewSyscall() *Syscall {
	sys := new(Syscall)
	sys.ctor()
	return sys
}

func (sys *Syscall) ctor() {
	sys.fcntl.ctor()
	sys.futex.ctor()
	sys.signal.ctor()
}

func (sys *Syscall) Close() error {
	sys.signal.dtor()
	sys.futex.dtor()
	sys.fcntl.dtor()
	return nil
}

func (sys *Syscall) Get(nr linux.NR) func(linux.Context, ...uint64) uint64 {
	switch nr {
	case linux.NR_reject:
		return sys.Reject
	case linux.NR_ignore:
		return sys.Ignore
	case linux.NR_dup3:
		return sys.Emulate_dup3
	case linux.NR_fcntl:
		return sys.Emulate_fcntl
	case linux.NR_faccessat:
		return sys.Emulate_faccessat
	case linux.NR_open:
		return sys.Emulate_open
	case linux.NR_openat:
		return sys.Emulate_openat
	case linux.NR_close:
		return sys.Emulate_close
	case linux.NR_pipe2:
		return sys.Emulate_pipe2
	case linux.NR_lseek:
		return sys.Emulate_lseek
	case linux.NR_read:
		return sys.Emulate_read
	case linux.NR_write:
		return sys.Emulate_write
	case linux.NR_writev:
		return sys.Emulate_writev
	case linux.NR_readlinkat:
		return sys.Emulate_readlinkat
	case linux.NR_fstatat64:
		return sys.Emulate_fstatat64
	case linux.NR_fstat64:
		return sys.Emulate_fstat64
	case linux.NR_exit, linux.NR_exit_group:
		return sys.Emulate_exit
	case linux.NR_futex:
		return sys.Emulate_futex
	case linux.NR_clock_gettime:
		return sys.Emulate_clock_gettime
	case linux.NR_sigaltstack:
		return sys.Ignore
	case linux.NR_rt_sigaction:
		return sys.Emulate_rt_sigaction
	case linux.NR_rt_sigprocmask:
		return sys.Emulate_rt_sigprocmask
	case linux.NR_prctl:
		return sys.Emulate_prctl
	case linux.NR_gettimeofday:
		return sys.Emulate_gettimeofday
	case linux.NR_getpid:
		return sys.Emulate_getpid
	case linux.NR_getuid, linux.NR_geteuid:
		return sys.Ignore
	case linux.NR_gettid:
		return sys.Emulate_gettid
	case linux.NR_sysinfo:
		return sys.Emulate_sysinfo
	case linux.NR_socket:
		return sys.Emulate_socket
	case linux.NR_munmap:
		return sys.Emulate_munmap
	case linux.NR_clone:
		return sys.Emulate_clone
	case linux.NR_execve:
		return sys.Emulate_execve
	case linux.NR_mmap:
		return sys.Emulate_mmap
	case linux.NR_mmap2:
		return sys.Emulate_mmap2
	case linux.NR_mprotect:
		return sys.Emulate_mprotect
	case linux.NR_madvise:
		return sys.Reject
	case linux.NR_rt_tgsigqueueinfo:
		return sys.Emulate_rt_tgsigqueueinfo
	case linux.NR_getrandom:
		return sys.Emulate_getrandom
	}
	return nil
}

func (sys *Syscall) Reject(ctx linux.Context, args ...uint64) uint64 {
	ctx.SetErrno(linux.ENOSYS)
	return math.MaxUint64
}

func (sys *Syscall) Ignore(ctx linux.Context, args ...uint64) uint64 {
	return 0
}

func (sys *Syscall) Emulate_dup3(ctx linux.Context, args ...uint64) uint64 {
	r := sys.fcntl.dup3(ctx, uint32(args[0]), uint32(args[1]), int32(args[2]))
	return uint64(r)
}

func (sys *Syscall) Emulate_fcntl(ctx linux.Context, args ...uint64) uint64 {
	r := sys.fcntl.fcntl(ctx, uint32(args[0]), uint32(args[1]), args[2])
	return uint64(r)
}

func (sys *Syscall) Emulate_faccessat(ctx linux.Context, args ...uint64) uint64 {
	r := sys.fcntl.faccessat(ctx, int32(args[0]), args[1], int32(args[2]))
	return uint64(r)
}

func (sys *Syscall) Emulate_open(ctx linux.Context, args ...uint64) uint64 {
	r := sys.fcntl.open(ctx, args[0], int32(args[1]), int32(args[2]))
	return uint64(r)
}

func (sys *Syscall) Emulate_openat(ctx linux.Context, args ...uint64) uint64 {
	r := sys.fcntl.openat(ctx, int32(args[0]), args[1], int32(args[2]), int32(args[3]))
	return uint64(r)
}

func (sys *Syscall) Emulate_close(ctx linux.Context, args ...uint64) uint64 {
	r := sys.fcntl.close(ctx, uint32(args[0]))
	return uint64(r)
}

func (sys *Syscall) Emulate_pipe2(ctx linux.Context, args ...uint64) uint64 {
	r := sys.fcntl.pipe2(ctx, args[0], int32(args[1]))
	return uint64(r)
}

func (sys *Syscall) Emulate_lseek(ctx linux.Context, args ...uint64) uint64 {
	r := sys.fcntl.lseek(ctx, uint32(args[0]), off_t(args[1]), int32(args[2]))
	return uint64(r)
}

func (sys *Syscall) Emulate_read(ctx linux.Context, args ...uint64) uint64 {
	r := sys.fcntl.read(ctx, uint32(args[0]), args[1], size_t(args[2]))
	return uint64(r)
}

func (sys *Syscall) Emulate_write(ctx linux.Context, args ...uint64) uint64 {
	r := sys.fcntl.write(ctx, uint32(args[0]), args[1], size_t(args[2]))
	return uint64(r)
}

func (sys *Syscall) Emulate_writev(ctx linux.Context, args ...uint64) uint64 {
	r := sys.fcntl.writev(ctx, uint32(args[0]), args[1], int32(args[2]))
	return uint64(r)
}

func (sys *Syscall) Emulate_readlinkat(ctx linux.Context, args ...uint64) uint64 {
	r := sys.fcntl.readlinkat(ctx, int32(args[0]), args[1], args[2], size_t(args[3]))
	return uint64(r)
}

func (sys *Syscall) Emulate_fstatat64(ctx linux.Context, args ...uint64) uint64 {
	var r int32
	switch ctx.Debugger().Emulator().Arch() {
	case emulator.ARCH_ARM, emulator.ARCH_X86:
		r = sys.fcntl.fstatat3264(ctx, int32(args[0]), args[1], args[2], int32(args[3]))
	case emulator.ARCH_ARM64, emulator.ARCH_X86_64:
		r = sys.fcntl.fstatat64(ctx, int32(args[0]), args[1], args[2], int32(args[3]))
	default:
		ctx.SetErrno(linux.ENOSYS)
		r = -1
	}
	return uint64(r)
}

func (sys *Syscall) Emulate_fstat64(ctx linux.Context, args ...uint64) uint64 {
	var r int32
	switch ctx.Debugger().Emulator().Arch() {
	case emulator.ARCH_ARM, emulator.ARCH_X86:
		r = sys.fcntl.fstat3264(ctx, uint32(args[0]), args[1])
	case emulator.ARCH_ARM64, emulator.ARCH_X86_64:
		r = sys.fcntl.fstat64(ctx, uint32(args[0]), args[1])
	default:
		ctx.SetErrno(linux.ENOSYS)
		r = -1
	}
	return uint64(r)
}

func (sys *Syscall) Emulate_exit(ctx linux.Context, args ...uint64) uint64 {
	panic("syscall exit")
}

func (sys *Syscall) Emulate_futex(ctx linux.Context, args ...uint64) uint64 {
	r := sys.futex.futex(ctx, args[0], int32(args[1]), uint32(args[2]), args[3], args[4], uint32(args[5]))
	return uint64(r)
}

func (sys *Syscall) Emulate_clock_gettime(ctx linux.Context, args ...uint64) uint64 {
	r := sys.clock_gettime(ctx, clockid_t(args[0]), args[1])
	return uint64(r)
}

func (sys *Syscall) Emulate_rt_sigaction(ctx linux.Context, args ...uint64) uint64 {
	r := sys.signal.rt_sigaction(ctx, int32(args[0]), args[1], args[2], size_t(args[3]))
	return uint64(r)
}

func (sys *Syscall) Emulate_rt_sigprocmask(ctx linux.Context, args ...uint64) uint64 {
	r := sys.signal.rt_sigprocmask(ctx, int32(args[0]), args[1], args[2], size_t(args[3]))
	return uint64(r)
}

func (sys *Syscall) Emulate_prctl(ctx linux.Context, args ...uint64) uint64 {
	r := sys.prctl.prctl(ctx, int32(args[0]), ulong_t(args[1]), ulong_t(args[2]), ulong_t(args[3]), ulong_t(args[4]))
	return uint64(r)
}

func (sys *Syscall) Emulate_gettimeofday(ctx linux.Context, args ...uint64) uint64 {
	r := sys.gettimeofday(ctx, args[0], args[1])
	return uint64(r)
}

func (sys *Syscall) Emulate_getpid(ctx linux.Context, args ...uint64) uint64 {
	r := sys.prctl.getpid(ctx)
	return uint64(r)
}

func (sys *Syscall) Emulate_gettid(ctx linux.Context, args ...uint64) uint64 {
	r := sys.prctl.gettid(ctx)
	return uint64(r)
}

func (sys *Syscall) Emulate_sysinfo(ctx linux.Context, args ...uint64) uint64 {
	r := sys.sysinfo(ctx, args[0])
	return uint64(r)
}

func (sys *Syscall) Emulate_socket(ctx linux.Context, args ...uint64) uint64 {
	r := sys.socket.socket(ctx, int32(args[0]), int32(args[1]), int32(args[2]))
	return uint64(r)
}

func (sys *Syscall) Emulate_munmap(ctx linux.Context, args ...uint64) uint64 {
	r := sys.mman.munmap(ctx, args[0], size_t(args[1]))
	return uint64(r)
}

func (sys *Syscall) Emulate_clone(ctx linux.Context, args ...uint64) uint64 {
	r := sys.sched.clone(ctx, int32(args[0]), args[1], args[2], args[3], args[4])
	return uint64(r)
}

func (sys *Syscall) Emulate_execve(ctx linux.Context, args ...uint64) uint64 {
	r := sys.sched.execve(ctx, args[0], args[1], args[2])
	return uint64(r)
}

func (sys *Syscall) Emulate_mmap(ctx linux.Context, args ...uint64) uint64 {
	r := sys.mman.mmap(ctx, args[0], size_t(args[1]), emulator.MemProt(args[2]), int32(args[3]), int32(args[4]), off_t(args[5]))
	return r
}

func (sys *Syscall) Emulate_mmap2(ctx linux.Context, args ...uint64) uint64 {
	r := sys.mman.mmap2(ctx, args[0], size_t(args[1]), emulator.MemProt(args[2]), int32(args[3]), int32(args[4]), size_t(args[5]))
	return r
}

func (sys *Syscall) Emulate_mprotect(ctx linux.Context, args ...uint64) uint64 {
	r := sys.mman.mprotect(ctx, args[0], size_t(args[1]), emulator.MemProt(args[2]))
	return uint64(r)
}

func (sys *Syscall) Emulate_rt_tgsigqueueinfo(ctx linux.Context, args ...uint64) uint64 {
	r := sys.signal.rt_tgsigqueueinfo(ctx, int32(args[0]), int32(args[1]), int32(args[2]), args[3])
	return uint64(r)
}

func (sys *Syscall) Emulate_getrandom(ctx linux.Context, args ...uint64) uint64 {
	r := sys.getrandom(ctx, args[0], size_t(args[1]), uint32(args[2]))
	return uint64(r)
}
