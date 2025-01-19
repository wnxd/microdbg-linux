package kernel

import (
	"crypto/rand"
	"io"

	linux "github.com/wnxd/microdbg-linux"
)

func (*Syscall) getrandom(ctx linux.Context, buf emuptr, count size_t, flags uint32) ssize_t {
	const (
		GRND_RANDOM   = 0x0001
		GRND_NONBLOCK = 0x0002
	)
	ptr := ctx.ToPointer(buf)
	if flags&GRND_RANDOM != 0 {
		n, err := io.CopyN(io.NewOffsetWriter(ptr, 0), rand.Reader, int64(count))
		if err != nil {
			ctx.SetErrno(linux.EAGAIN)
			return -1
		}
		return ssize_t(n)
	} else {
		data := make([]byte, min(count, 256))
		n, err := rand.Read(data)
		if err != nil {
			ctx.SetErrno(linux.EAGAIN)
			return -1
		}
		ptr.MemWrite(data[:n])
		return ssize_t(n)
	}
}
