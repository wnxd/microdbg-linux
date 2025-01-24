package kernel

import linux "github.com/wnxd/microdbg-linux"

const RLIMIT_STACK = 3

type rlimit struct {
	rlim_cur ulong_t
	rlim_max ulong_t
}

type resource struct {
}

func (r *resource) getrlimit(ctx linux.Context, resource int32, rlim emuptr) int32 {
	switch resource {
	case RLIMIT_STACK:
		dbg := ctx.Debugger()
		dbg.MemWrite(rlim, rlimit{
			rlim_cur: ulong_t(dbg.StackSize()),
			rlim_max: ulong_t(dbg.StackSize()),
		})
	}
	return 0
}

func (r *resource) setrlimit(ctx linux.Context, resource int32, rlim emuptr) int32 {
	return 0
}
