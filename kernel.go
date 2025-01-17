package linux

type Syscall interface {
	Get(nr NR) func(ctx Context, args ...uint64) uint64
}

type Kernel interface {
	NR(no uint64) NR
	Syscall() Syscall
	Errno() Errno
	SetErrno(err Errno)
}
