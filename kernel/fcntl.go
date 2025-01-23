package kernel

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sync"
	"unsafe"

	linux "github.com/wnxd/microdbg-linux"
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
	iov_base uintptr
	iov_len  size_t
}

type stat3264 struct {
	st_dev     uint64
	_          uint32
	__st_ino   ino_t
	st_mode    mode_t
	st_nlink   nlink_t
	st_uid     uid_t
	st_gid     gid_t
	st_rdev    uint64
	_          uint32
	st_size    int64
	st_blksize ulong_t
	st_blocks  uint64
	st_atim    timespec
	st_mtim    timespec
	st_ctim    timespec
	st_ino     uint64
}

type stat64 struct {
	st_dev     dev_t
	st_ino     ino_t
	st_mode    mode_t
	st_nlink   nlink_t
	st_uid     uid_t
	st_gid     gid_t
	st_rdev    dev_t
	_          ulong_t
	st_size    off_t
	st_blksize int32
	_          int32
	st_blocks  long_t
	st_atim    timespec
	st_mtim    timespec
	st_ctim    timespec
	_          uint32
	_          uint32
}

var (
	_ = stat3264{}.st_dev
	_ = stat3264{}.__st_ino
	_ = stat3264{}.st_nlink
	_ = stat3264{}.st_uid
	_ = stat3264{}.st_gid
	_ = stat3264{}.st_rdev
	_ = stat3264{}.st_blksize
	_ = stat3264{}.st_blocks
	_ = stat3264{}.st_ino
	_ = stat64{}.st_dev
	_ = stat64{}.st_ino
	_ = stat64{}.st_nlink
	_ = stat64{}.st_uid
	_ = stat64{}.st_gid
	_ = stat64{}.st_rdev
	_ = stat64{}.st_blksize
	_ = stat64{}.st_blocks
)

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

func (f *fcntl) faccessat(ctx linux.Context, dfd int32, filename emuptr, mode int32) int32 {
	path, err := ctx.ToPointer(filename).MemReadString()
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

func (f *fcntl) open(ctx linux.Context, filename emuptr, flags, mode int32) int32 {
	path, err := ctx.ToPointer(filename).MemReadString()
	if err != nil {
		ctx.SetErrno(linux.ENOENT)
		return -1
	}
	dbg := ctx.Debugger()
	file, err := dbg.OpenFile(path, toFileFlag(flags), fs.FileMode(mode))
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

func (f *fcntl) openat(ctx linux.Context, dfd int32, filename emuptr, flags, mode int32) int32 {
	path, err := ctx.ToPointer(filename).MemReadString()
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

func (f *fcntl) pipe2(ctx linux.Context, fildes emuptr, flags int32) int32 {
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
	dbg.MemWrite(fildes, [2]int32{int32(rfd), int32(wfd)})
	return 0
}

func (f *fcntl) lseek(ctx linux.Context, fd uint32, offset off_t, whence int32) off_t {
	file, err := ctx.Debugger().GetFile(int(fd))
	if err != nil {
		ctx.SetErrno(linux.EBADF)
		return -1
	}
	if seek, ok := file.(io.Seeker); ok {
		off, err := seek.Seek(int64(offset), int(whence))
		if err != nil {
			ctx.SetErrno(linux.EINVAL)
			return -1
		}
		return off_t(off)
	}
	ctx.SetErrno(linux.EINVAL)
	return -1
}

func (f *fcntl) read(ctx linux.Context, fd uint32, buf emuptr, count size_t) ssize_t {
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
	n, err := io.CopyN(io.NewOffsetWriter(ctx.ToPointer(buf), 0), r, int64(count))
	if err != nil {
		ctx.SetErrno(linux.EIO)
		return -1
	}
	return ssize_t(n)
}

func (f *fcntl) write(ctx linux.Context, fd uint32, buf emuptr, count size_t) ssize_t {
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
	n, err := io.Copy(w, io.NewSectionReader(ctx.ToPointer(buf), 0, int64(count)))
	if err != nil {
		ctx.SetErrno(linux.EIO)
		return -1
	}
	return ssize_t(n)
}

func (f *fcntl) writev(ctx linux.Context, fd uint32, iov emuptr, iovcnt int32) ssize_t {
	dbg := ctx.Debugger()
	file, err := dbg.GetFile(int(fd))
	if err != nil {
		ctx.SetErrno(linux.EBADF)
		return -1
	}
	w, ok := file.(io.Writer)
	if !ok {
		ctx.SetErrno(linux.EINTR)
		return -1
	}
	arr := make([]iovec, iovcnt)
	dbg.MemExtract(iov, arr)
	var n ssize_t
	for i := range arr {
		ptr := ctx.ToPointer(emuptr(arr[i].iov_base))
		m, err := io.Copy(w, io.NewSectionReader(ptr, 0, int64(arr[i].iov_len)))
		if err != nil {
			ctx.SetErrno(linux.EIO)
			return -1
		}
		n += ssize_t(m)
	}
	return n
}

func (f *fcntl) readlinkat(ctx linux.Context, dfd int32, filename, buf emuptr, bufsiz size_t) ssize_t {
	path, err := ctx.ToPointer(filename).MemReadString()
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
	link, err := dir.Readlink(path)
	if err != nil {
		ctx.SetErrno(linux.EINVAL)
		return -1
	}
	size := min(uint64(len(link)), uint64(bufsiz))
	ctx.ToPointer(buf).MemWritePtr(size, unsafe.Pointer(unsafe.StringData(link)))
	return ssize_t(size)
}

func (f *fcntl) fstatat3264(ctx linux.Context, dfd int32, filename, statbuf emuptr, flag int32) int32 {
	path, err := ctx.ToPointer(filename).MemReadString()
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
	var stat stat3264
	mode := info.Mode()
	stat.st_mode = mode_t(mode.Perm())
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
	ts := timespec{
		tv_sec:  time_t(nano / 1e9),
		tv_nsec: long_t(nano % 1e9),
	}
	stat.st_atim = ts
	stat.st_mtim = ts
	stat.st_ctim = ts
	dbg.MemWrite(statbuf, stat)
	return 0
}

func (f *fcntl) fstatat64(ctx linux.Context, dfd int32, filename, statbuf emuptr, flag int32) int32 {
	path, err := ctx.ToPointer(filename).MemReadString()
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
	var stat stat64
	mode := info.Mode()
	stat.st_mode = mode_t(mode.Perm())
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
	stat.st_size = off_t(info.Size())
	nano := info.ModTime().UnixNano()
	ts := timespec{
		tv_sec:  time_t(nano / 1e9),
		tv_nsec: long_t(nano % 1e9),
	}
	stat.st_atim = ts
	stat.st_mtim = ts
	stat.st_ctim = ts
	dbg.MemWrite(statbuf, stat)
	return 0
}

func (f *fcntl) fstat3264(ctx linux.Context, fd uint32, statbuf emuptr) int32 {
	dbg := ctx.Debugger()
	file, err := dbg.GetFile(int(fd))
	if err != nil {
		ctx.SetErrno(linux.EBADF)
		return -1
	}
	info, err := file.Stat()
	if err != nil {
		ctx.SetErrno(linux.ENOENT)
		return -1
	}
	var stat stat3264
	mode := info.Mode()
	stat.st_mode = mode_t(mode.Perm())
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
	ts := timespec{
		tv_sec:  time_t(nano / 1e9),
		tv_nsec: long_t(nano % 1e9),
	}
	stat.st_atim = ts
	stat.st_mtim = ts
	stat.st_ctim = ts
	dbg.MemWrite(statbuf, stat)
	return 0
}

func (f *fcntl) fstat64(ctx linux.Context, fd uint32, statbuf emuptr) int32 {
	dbg := ctx.Debugger()
	file, err := dbg.GetFile(int(fd))
	if err != nil {
		ctx.SetErrno(linux.EBADF)
		return -1
	}
	info, err := file.Stat()
	if err != nil {
		ctx.SetErrno(linux.ENOENT)
		return -1
	}
	var stat stat64
	mode := info.Mode()
	stat.st_mode = mode_t(mode.Perm())
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
	stat.st_size = off_t(info.Size())
	nano := info.ModTime().UnixNano()
	ts := timespec{
		tv_sec:  time_t(nano / 1e9),
		tv_nsec: long_t(nano % 1e9),
	}
	stat.st_atim = ts
	stat.st_mtim = ts
	stat.st_ctim = ts
	dbg.MemWrite(statbuf, stat)
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
