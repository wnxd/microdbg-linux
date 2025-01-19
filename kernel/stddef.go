package kernel

type emuptr = uint64
type emutptr[T any] emuptr

type long_t int
type ulong_t uint
type size_t ulong_t
type ssize_t long_t
type time_t long_t
type suseconds_t long_t
type clockid_t int32
type off_t long_t
type dev_t ulong_t
type ino_t ulong_t
type mode_t uint32
type nlink_t uint32
type uid_t uint32
type gid_t uint32
type pid_t int32

const emunullptr = emuptr(0)
