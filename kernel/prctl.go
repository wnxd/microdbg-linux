package kernel

import (
	"os"

	linux "github.com/wnxd/microdbg-linux"
	"github.com/wnxd/microdbg/debugger"
)

type prctl struct {
}

func (k *prctl) prctl(ctx linux.Context, option int32, arg1, arg2, arg3, arg4 ulong_t) int32 {
	const (
		PR_SET_VMA = 0x53564d41
	)

	switch option {
	case PR_SET_VMA:
		return 0
	}
	ctx.SetErrno(linux.EINVAL)
	return -1
}

func (k *prctl) getpid(ctx debugger.Context) pid_t {
	return pid_t(os.Getpid())
}

func (k *prctl) gettid(ctx debugger.Context) pid_t {
	return pid_t(ctx.TaskID())
}
