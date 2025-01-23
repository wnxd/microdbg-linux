package kernel

import (
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
	linux "github.com/wnxd/microdbg-linux"
)

type sysinfo struct {
	uptime    long_t
	loads     [3]ulong_t
	totalram  ulong_t
	freeram   ulong_t
	sharedram ulong_t
	bufferram ulong_t
	totalswap ulong_t
	freeswap  ulong_t
	procs     uint16
	_f        [22]byte
}

var (
	_ = sysinfo{}.loads
	_ = sysinfo{}._f
)

func (*Syscall) sysinfo(ctx linux.Context, info emuptr) int32 {
	uptime, err := host.Uptime()
	if err != nil {
		ctx.SetErrno(linux.EINVAL)
		return -1
	}
	vm, err := mem.VirtualMemory()
	if err != nil {
		ctx.SetErrno(linux.EINVAL)
		return -1
	}
	sm, err := mem.SwapMemory()
	if err != nil {
		ctx.SetErrno(linux.EINVAL)
		return -1
	}
	pids, err := process.Pids()
	if err != nil {
		ctx.SetErrno(linux.EINVAL)
		return -1
	}
	_, err = ctx.Debugger().MemWrite(info, sysinfo{
		uptime:    long_t(uptime),
		totalram:  ulong_t(vm.Total),
		freeram:   ulong_t(vm.Free),
		sharedram: ulong_t(vm.Shared),
		bufferram: ulong_t(vm.Buffers),
		totalswap: ulong_t(sm.Total),
		freeswap:  ulong_t(sm.Free),
		procs:     uint16(len(pids)),
	})
	if err != nil {
		ctx.SetErrno(linux.EINVAL)
		return -1
	}
	return 0
}
