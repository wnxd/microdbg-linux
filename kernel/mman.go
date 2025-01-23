package kernel

import (
	"io"
	"math"

	linux "github.com/wnxd/microdbg-linux"
	"github.com/wnxd/microdbg/emulator"
	"github.com/wnxd/microdbg/filesystem"
)

const PAGE_SIZE = 4096

type mman struct {
}

func (k *mman) munmap(ctx linux.Context, addr emuptr, len size_t) int32 {
	err := ctx.Debugger().MapFree(addr, uint64(len))
	if err != nil {
		ctx.SetErrno(linux.EINVAL)
		return -1
	}
	return 0
}

func (k *mman) mmap(ctx linux.Context, addr emuptr, len size_t, prot emulator.MemProt, flags, fd int32, offset off_t) emuptr {
	const (
		MAP_FAILED    = math.MaxUint64
		MAP_SHARED    = 0x01
		MAP_PRIVATE   = 0x02
		MAP_FIXED     = 0x10
		MAP_ANONYMOUS = 0x20
	)

	dbg := ctx.Debugger()
	var f filesystem.ReadFile
	if flags&MAP_ANONYMOUS == 0 && fd >= 0 {
		file, err := dbg.GetFile(int(fd))
		if err != nil {
			ctx.SetErrno(linux.EBADF)
			return MAP_FAILED
		}
		var ok bool
		if f, ok = file.(filesystem.ReadFile); !ok {
			ctx.SetErrno(linux.ENODEV)
			return MAP_FAILED
		} else if offset != 0 {
			var err error
			if seek, ok := f.(io.Seeker); ok {
				_, err = seek.Seek(int64(offset), io.SeekStart)
			} else {
				_, err = io.ReadFull(f, make([]byte, offset))
			}
			if err != nil {
				ctx.SetErrno(linux.ENODEV)
				return MAP_FAILED
			}
		}
	}
	if flags&MAP_FIXED != 0 {
		dbg.MemUnmap(addr, uint64(len))
		region, err := dbg.MemMap(addr, uint64(len), prot)
		if err != nil {
			ctx.SetErrno(linux.EINVAL)
			return MAP_FAILED
		}
		addr = region.Addr
	} else {
		region, err := dbg.MapAlloc(uint64(len), prot)
		if err != nil {
			ctx.SetErrno(linux.EINVAL)
			return MAP_FAILED
		}
		addr = region.Addr
	}
	if f == nil {
		return addr
	}
	_, err := io.CopyN(io.NewOffsetWriter(ctx.ToPointer(addr), 0), f, int64(len))
	if err != nil {
		dbg.MapFree(addr, uint64(len))
		ctx.SetErrno(linux.ENODEV)
		return MAP_FAILED
	}
	return addr
}

func (k *mman) mmap2(ctx linux.Context, addr emuptr, len size_t, prot emulator.MemProt, flags, fd int32, count size_t) emuptr {
	return k.mmap(ctx, addr, len, prot, flags, fd, off_t(count*PAGE_SIZE))
}

func (k *mman) mprotect(ctx linux.Context, start emuptr, len size_t, prot emulator.MemProt) int32 {
	err := ctx.Debugger().MemProtect(start, uint64(len), prot)
	if err != nil {
		ctx.SetErrno(linux.EINVAL)
		return -1
	}
	return 0
}
