package kernel_lp64

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sync"
	"unsafe"

	linux "github.com/wnxd/microdbg-linux"
	"github.com/wnxd/microdbg/emulator"
	"github.com/wnxd/microdbg/filesystem"
)

const (
	AT_FDCWD = -100

	S_IFIFO  = 0x1000
	S_IFCHR  = 0x2000
	S_IFDIR  = 0x4000
	S_IFBLK  = 0x6000
	S_IFREG  = 0x8000
	S_IFLNK  = 0xA000
	S_IFSOCK = 0xC000
)

type iovec struct {
	iov_base emulator.Uintptr64
	iov_len  uint64
}

type stat struct {
	st_dev     uint64
	st_ino     uint64
	st_mode    uint32
	st_nlink   uint32
	st_uid     uint32
	st_gid     uint32
	st_rdev    uint64
	__pad1     uint64
	st_size    int64
	st_blksize int32
	__pad2     int32
	st_blocks  int64
	st_atim    timespec
	st_mtim    timespec
	st_ctim    timespec
	__unused4  uint32
	__unused5  uint32
}

type fcntl struct {
	rw    sync.RWMutex
	flags map[int]int32
}

func (f *fcntl) ctor() {
	f.flags = make(map[int]int32)
}

func (f *fcntl) dtor() {
	f.flags = nil
}

func (f *fcntl) dup3(ctx linux.Context, oldfd, newfd uint32, flags int32) int32 {
	err := ctx.Debugger().Dup2File(int(oldfd), int(newfd))
	if err != nil {
		ctx.SetErrno(linux.EBADF)
		return -1
	}
	f.rw.Lock()
	f.flags[int(newfd)] = flags
	f.rw.Unlock()
	return int32(newfd)
}

func (f *fcntl) fcntl(ctx linux.Context, fd, cmd uint32, arg uint64) int32 {
	const (
		F_DUPFD = iota
		F_GETFD
		F_SETFD
		F_GETFL
		F_SETFL
		F_GETLK
		F_SETLK
		F_SETLKW
		F_SETOWN
		F_GETOWN
		F_SETSIG
		F_GETSIG
		F_GETLK64
		F_SETLK64
		F_SETLKW64

		O_CLOEXEC = 0x80000
	)

	dbg := ctx.Debugger()
	_, err := dbg.GetFile(int(fd))
	if err != nil {
		ctx.SetErrno(linux.EBADF)
		return -1
	}
	switch cmd {
	case F_DUPFD:
		newfd, err := dbg.DupFile(int(fd))
		if err != nil {
			ctx.SetErrno(linux.EBADF)
			return -1
		}
		f.rw.Lock()
		f.flags[newfd] = f.flags[int(fd)]
		f.rw.Unlock()
		return int32(newfd)
	case F_GETFD:
		f.rw.RLock()
		hasExec := f.flags[int(fd)]&O_CLOEXEC != 0
		f.rw.RUnlock()
		if hasExec {
			return 1
		}
		return 0
	case F_SETFD:
		f.rw.Lock()
		f.flags[int(fd)] |= O_CLOEXEC
		f.rw.Unlock()
		return 0
	case F_GETFL:
		f.rw.RLock()
		flag := f.flags[int(fd)]
		f.rw.RUnlock()
		return flag
	case F_SETFL:
		f.rw.Lock()
		f.flags[int(fd)] = int32(arg)
		f.rw.Unlock()
		return 0
	case F_GETLK, F_GETLK64:
		return 0
	case F_SETLK, F_SETLKW, F_SETLK64, F_SETLKW64:
		return 0
	}
	panic(fmt.Errorf("fcntl: %d %w", cmd, errors.ErrUnsupported))
}

func (f *fcntl) faccessat(ctx linux.Context, dfd int32, filename emulator.Pointer, mode int32) int32 {
	path, err := filename.MemReadString()
	if err != nil {
		ctx.SetErrno(linux.ENOENT)
		return -1
	}
	dbg := ctx.Debugger()
	var dir filesystem.FS
	if dfd != AT_FDCWD {
		file, err := dbg.GetFile(int(dfd))
		if err != nil {
			ctx.SetErrno(linux.EBADF)
			return -1
		}
		var ok bool
		dir, ok = file.(filesystem.FS)
		if !ok {
			ctx.SetErrno(linux.ENOTDIR)
			return -1
		}
	} else {
		dir = dbg.GetFS()
	}
	file, err := dir.OpenFile(path, filesystem.O_RDONLY, fs.FileMode(mode))
	if err != nil {
		ctx.SetErrno(linux.ENOENT)
		return -1
	}
	file.Close()
	return 0
}

func (f *fcntl) openat(ctx linux.Context, dfd int32, filename emulator.Pointer, flags, mode int32) int32 {
	path, err := filename.MemReadString()
	if err != nil {
		ctx.SetErrno(linux.ENOENT)
		return -1
	}
	dbg := ctx.Debugger()
	var dir filesystem.FS
	if dfd != AT_FDCWD {
		file, err := dbg.GetFile(int(dfd))
		if err != nil {
			ctx.SetErrno(linux.EBADF)
			return -1
		}
		var ok bool
		dir, ok = file.(filesystem.FS)
		if !ok {
			ctx.SetErrno(linux.ENOTDIR)
			return -1
		}
	} else {
		dir = dbg.GetFS()
	}
	file, err := dir.OpenFile(path, toFileFlag(flags), fs.FileMode(mode))
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			ctx.SetErrno(linux.EEXIST)
			return -1
		}
		ctx.SetErrno(linux.ENOENT)
		return -1
	}
	fd := dbg.CreateFileDescriptor(file)
	f.rw.Lock()
	f.flags[fd] = flags
	f.rw.Unlock()
	return int32(fd)
}

func (f *fcntl) close(ctx linux.Context, fd uint32) int32 {
	file, err := ctx.Debugger().CloseFileDescriptor(int(fd))
	if err != nil {
		ctx.SetErrno(linux.EBADF)
		return -1
	}
	file.Close()
	f.rw.Lock()
	delete(f.flags, int(fd))
	f.rw.Unlock()
	return 0
}

func (f *fcntl) pipe2(ctx linux.Context, fildes emulator.Pointer, flags int32) int32 {
	r, w, err := os.Pipe()
	if err != nil {
		ctx.SetErrno(linux.EMFILE)
		return -1
	}
	dbg := ctx.Debugger()
	rfd := dbg.CreateFileDescriptor(r)
	wfd := dbg.CreateFileDescriptor(w)
	f.rw.Lock()
	f.flags[rfd] = flags
	f.flags[wfd] = flags
	f.rw.Unlock()
	fildes.MemWritePtr(4, unsafe.Pointer(&rfd))
	fildes.Add(4).MemWritePtr(4, unsafe.Pointer(&wfd))
	return 0
}

func (f *fcntl) lseek(ctx linux.Context, fd uint32, offset int64, whence int32) int64 {
	file, err := ctx.Debugger().GetFile(int(fd))
	if err != nil {
		ctx.SetErrno(linux.EBADF)
		return -1
	}
	if seek, ok := file.(io.Seeker); ok {
		offset, err = seek.Seek(offset, int(whence))
		if err != nil {
			ctx.SetErrno(linux.EINVAL)
			return -1
		}
		return offset
	}
	panic("lseek")
}

func (f *fcntl) read(ctx linux.Context, fd uint32, buf emulator.Pointer, count uint64) int64 {
	file, err := ctx.Debugger().GetFile(int(fd))
	if err != nil {
		ctx.SetErrno(linux.EBADF)
		return -1
	}
	r, ok := file.(io.Reader)
	if !ok {
		ctx.SetErrno(linux.EINTR)
		return -1
	}
	n, err := io.CopyN(io.NewOffsetWriter(buf, 0), r, int64(count))
	if err != nil {
		ctx.SetErrno(linux.EIO)
		return -1
	}
	return n
}

func (f *fcntl) write(ctx linux.Context, fd uint32, buf emulator.Pointer, count uint64) int64 {
	file, err := ctx.Debugger().GetFile(int(fd))
	if err != nil {
		ctx.SetErrno(linux.EBADF)
		return -1
	}
	w, ok := file.(io.Writer)
	if !ok {
		ctx.SetErrno(linux.EINTR)
		return -1
	}
	n, err := io.Copy(w, io.NewSectionReader(buf, 0, int64(count)))
	if err != nil {
		ctx.SetErrno(linux.EIO)
		return -1
	}
	return n
}

func (f *fcntl) writev(ctx linux.Context, fd uint32, iov emulator.Pointer, iovcnt uint64) int64 {
	file, err := ctx.Debugger().GetFile(int(fd))
	if err != nil {
		ctx.SetErrno(linux.EBADF)
		return -1
	}
	w, ok := file.(io.Writer)
	if !ok {
		ctx.SetErrno(linux.EINTR)
		return -1
	}
	var n int64
	for i := uint64(0); i < iovcnt; i++ {
		var buf iovec
		iov.MemReadPtr(uint64(unsafe.Sizeof(buf)), unsafe.Pointer(&buf))
		iov = iov.Add(uint64(unsafe.Sizeof(buf)))
		ptr := ctx.ToPointer(buf.iov_base)
		m, err := io.Copy(w, io.NewSectionReader(ptr, 0, int64(buf.iov_len)))
		if err != nil {
			ctx.SetErrno(linux.EIO)
			return -1
		}
		n += m
	}
	return n
}

func (f *fcntl) readlinkat(ctx linux.Context, dfd int32, path, buf emulator.Pointer, bufsiz uint64) int64 {
	name, err := path.MemReadString()
	if err != nil {
		ctx.SetErrno(linux.ENOENT)
		return -1
	}
	dbg := ctx.Debugger()
	var dir filesystem.ReadlinkFS
	if dfd != AT_FDCWD {
		file, err := dbg.GetFile(int(dfd))
		if err != nil {
			ctx.SetErrno(linux.EBADF)
			return -1
		}
		var ok bool
		dir, ok = file.(filesystem.ReadlinkFS)
		if !ok {
			ctx.SetErrno(linux.ENOTDIR)
			return -1
		}
	} else {
		dir = dbg.GetFS().(filesystem.ReadlinkFS)
	}
	link, err := dir.Readlink(name)
	if err != nil {
		ctx.SetErrno(linux.EINVAL)
		return -1
	}
	size := min(uint64(len(link)), bufsiz)
	buf.MemWritePtr(size, unsafe.Pointer(unsafe.StringData(link)))
	return 0
}

func (f *fcntl) newfstatat(ctx linux.Context, dfd int32, filename, statbuf emulator.Pointer, flag int32) int32 {
	path, err := filename.MemReadString()
	if err != nil {
		ctx.SetErrno(linux.ENOENT)
		return -1
	}
	dbg := ctx.Debugger()
	var dir fs.FS
	if dfd != AT_FDCWD {
		file, err := dbg.GetFile(int(dfd))
		if err != nil {
			ctx.SetErrno(linux.EBADF)
			return -1
		}
		var ok bool
		dir, ok = file.(fs.FS)
		if !ok {
			ctx.SetErrno(linux.ENOTDIR)
			return -1
		}
	} else {
		dir = dbg.GetFS()
	}
	info, err := fs.Stat(dir, path)
	if err != nil {
		ctx.SetErrno(linux.ENOENT)
		return -1
	}
	var stat stat
	mode := info.Mode()
	stat.st_mode = uint32(mode.Perm())
	switch mode.Type() {
	case fs.ModeCharDevice:
		stat.st_mode |= S_IFCHR
	case fs.ModeDevice:
		stat.st_mode |= S_IFBLK
	case fs.ModeDir:
		stat.st_mode |= S_IFDIR
	case fs.ModeNamedPipe:
		stat.st_mode |= S_IFIFO
	case fs.ModeSymlink:
		stat.st_mode |= S_IFLNK
	case fs.ModeSocket:
		stat.st_mode |= S_IFSOCK
	default:
		stat.st_mode |= S_IFREG
	}
	stat.st_size = info.Size()
	nano := info.ModTime().UnixNano()
	ts := timespec{int64(nano / 1e9), int64(nano % 1e9)}
	stat.st_atim = ts
	stat.st_mtim = ts
	stat.st_ctim = ts
	statbuf.MemWritePtr(uint64(unsafe.Sizeof(stat)), unsafe.Pointer(&stat))
	return 0
}

func (f *fcntl) fstat(ctx linux.Context, fd uint32, statbuf emulator.Pointer) int32 {
	file, err := ctx.Debugger().GetFile(int(fd))
	if err != nil {
		ctx.SetErrno(linux.EBADF)
		return -1
	}
	info, err := file.Stat()
	if err != nil {
		ctx.SetErrno(linux.ENOENT)
		return -1
	}
	var stat stat
	mode := info.Mode()
	stat.st_mode = uint32(mode.Perm())
	switch mode.Type() {
	case fs.ModeCharDevice:
		stat.st_mode |= S_IFCHR
	case fs.ModeDevice:
		stat.st_mode |= S_IFBLK
	case fs.ModeDir:
		stat.st_mode |= S_IFDIR
	case fs.ModeNamedPipe:
		stat.st_mode |= S_IFIFO
	case fs.ModeSymlink:
		stat.st_mode |= S_IFLNK
	case fs.ModeSocket:
		stat.st_mode |= S_IFSOCK
	default:
		stat.st_mode |= S_IFREG
	}
	stat.st_size = info.Size()
	nano := info.ModTime().UnixNano()
	ts := timespec{int64(nano / 1e9), int64(nano % 1e9)}
	stat.st_atim = ts
	stat.st_mtim = ts
	stat.st_ctim = ts
	statbuf.MemWritePtr(uint64(unsafe.Sizeof(stat)), unsafe.Pointer(&stat))
	return 0
}

func toFileFlag(flags int32) filesystem.FileFlag {
	const (
		O_RDONLY = 0
		O_WRONLY = 1
		O_RDWR   = 2
		O_APPEND = 0x400
		O_CREAT  = 0x40
		O_EXCL   = 0x80
		O_SYNC   = 0x101000
		O_TRUNC  = 0x200
	)
	ff := filesystem.O_RDONLY
	if flags&O_WRONLY != 0 {
		ff = filesystem.O_WRONLY
	}
	if flags&O_RDWR != 0 {
		ff = filesystem.O_RDWR
	}
	if flags&O_APPEND != 0 {
		ff |= filesystem.O_APPEND
	}
	if flags&O_CREAT != 0 {
		ff |= filesystem.O_CREATE
	}
	if flags&O_EXCL != 0 {
		ff |= filesystem.O_EXCL
	}
	if flags&O_SYNC != 0 {
		ff |= filesystem.O_SYNC
	}
	if flags&O_TRUNC != 0 {
		ff |= filesystem.O_TRUNC
	}
	return ff
}
