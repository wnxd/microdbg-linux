package kernel_lp64

import (
	"crypto/rand"
	"io"

	linux "github.com/wnxd/microdbg-linux"
	"github.com/wnxd/microdbg/emulator"
)

func (*Syscall) getrandom(ctx linux.Context, buf emulator.Pointer, count uint64, flags uint32) int64 {
	const (
		GRND_RANDOM   = 0x0001
		GRND_NONBLOCK = 0x0002
	)
	if flags&GRND_RANDOM != 0 {
		n, err := io.CopyN(io.NewOffsetWriter(buf, 0), rand.Reader, int64(count))
		if err != nil {
			ctx.SetErrno(linux.EAGAIN)
			return -1
		}
		return n
	} else {
		data := make([]byte, min(count, 256))
		n, err := rand.Read(data)
		if err != nil {
			ctx.SetErrno(linux.EAGAIN)
			return -1
		}
		buf.MemWrite(data[:n])
		return int64(n)
	}
}
