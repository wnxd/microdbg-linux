package kernel_lp64

/*
#include <time.h>
*/
import "C"

import (
	"time"
	"unsafe"

	"github.com/wnxd/microdbg/emulator"
)

type timespec struct {
	tv_sec  int64
	tv_nsec int64
}

type timeval struct {
	tv_sec  int64
	tv_usec int64
}

type timezone struct {
	tz_minuteswest int32
	tz_dsttime     int32
}

func (sys *Syscall) clock_gettime(clock uint64, tp emulator.Pointer) int32 {
	var st C.struct_timespec
	r := C.clock_gettime(C.clockid_t(clock), &st)
	if r == 0 {
		tp.MemWritePtr(uint64(unsafe.Sizeof(st)), unsafe.Pointer(&st))
	}
	return int32(r)
}

func (sys *Syscall) gettimeofday(tv, tz emulator.Pointer) int32 {
	now := time.Now()
	stv := timeval{now.Unix(), int64(now.Nanosecond() / 1e3)}
	tv.MemWritePtr(uint64(unsafe.Sizeof(stv)), unsafe.Pointer(&stv))
	_, offset := now.Zone()
	stz := timezone{int32(offset / 60), 0}
	tz.MemReadPtr(uint64(unsafe.Sizeof(stz)), unsafe.Pointer(&stz))
	return 0
}
