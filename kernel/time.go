package kernel

/*
#include <time.h>
*/
import "C"

import (
	"time"

	linux "github.com/wnxd/microdbg-linux"
)

type timespec struct {
	tv_sec  time_t
	tv_nsec long_t
}

type timeval struct {
	tv_sec  time_t
	tv_usec suseconds_t
}

type timezone struct {
	tz_minuteswest int32
	tz_dsttime     int32
}

func (sys *Syscall) clock_gettime(ctx linux.Context, clock clockid_t, ts emuptr) int32 {
	var st C.struct_timespec
	r := C.clock_gettime(C.clockid_t(clock), &st)
	if r == 0 {
		ctx.Debugger().MemWrite(ts, timespec{
			tv_sec:  time_t(st.tv_sec),
			tv_nsec: long_t(st.tv_nsec),
		})
	}
	return int32(r)
}

func (sys *Syscall) gettimeofday(ctx linux.Context, tv, tz emuptr) int32 {
	dbg := ctx.Debugger()
	now := time.Now()
	dbg.MemWrite(tv, timeval{
		tv_sec:  time_t(now.Unix()),
		tv_usec: suseconds_t(now.Nanosecond() / 1e3),
	})
	_, offset := now.Zone()
	dbg.MemWrite(tz, timezone{
		tz_minuteswest: int32(offset / 60),
		tz_dsttime:     0,
	})
	return 0
}
