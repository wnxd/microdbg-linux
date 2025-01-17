package kernel_lp64

import (
	"os"

	linux "github.com/wnxd/microdbg-linux"
	"github.com/wnxd/microdbg/debugger"
)

type prctl struct {
}

func (k *prctl) prctl(ctx linux.Context, option int32, args ...uint64) int32 {
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

func (k *prctl) getpid(ctx debugger.Context) int32 {
	return int32(os.Getpid())
}

func (k *prctl) gettid(ctx debugger.Context) int32 {
	return int32(ctx.TaskID())
}
