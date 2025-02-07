package kernel

import (
	linux "github.com/wnxd/microdbg-linux"
	"github.com/wnxd/microdbg/filesystem"
)

func (sys *Syscall) ioctl(ctx linux.Context, fd, cmd uint32, arg emuptr) int32 {
	dbg := ctx.Debugger()
	file, err := dbg.GetFile(int(fd))
	if err != nil {
		ctx.SetErrno(linux.EBADF)
		return -1
	}
	if ctl, ok := file.(filesystem.ControlFile); ok {
		err := ctl.Control(int(cmd), arg)
		if err != nil {
			ctx.SetErrno(linux.EINVAL)
			return -1
		}
		return 0
	}
	ctx.SetErrno(linux.ENOTTY)
	return -1
}
