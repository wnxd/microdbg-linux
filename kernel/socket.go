package kernel

import (
	linux "github.com/wnxd/microdbg-linux"
)

type socket struct {
}

func (s *socket) socket(ctx linux.Context, domain, typ, protocol int32) int32 {
	panic("socket")
	ctx.SetErrno(linux.ENOSYS)
	return -1
}
