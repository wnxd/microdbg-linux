package kernel_lp64

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	linux "github.com/wnxd/microdbg-linux"
	"github.com/wnxd/microdbg/emulator"
)

type futexAwait struct {
	ch  chan uint32
	ref int
}

type futexBitAwait struct {
	ch  chan struct{}
	ref int
}

type futex struct {
	rw        sync.RWMutex
	awaits    map[emulator.Uintptr64]*futexAwait
	bitAwaits map[uint32]*futexBitAwait
}

func (f *futex) ctor() {
	f.awaits = make(map[emulator.Uintptr64]*futexAwait)
	f.bitAwaits = make(map[uint32]*futexBitAwait)
}

func (f *futex) dtor() {
	f.rw.Lock()
	for _, await := range f.awaits {
		close(await.ch)
	}
	for _, await := range f.bitAwaits {
		close(await.ch)
	}
	f.awaits = nil
	f.bitAwaits = nil
	f.rw.Unlock()
}

func (f *futex) futex(ctx linux.Context, uaddr emulator.Pointer, op int32, val uint32, utime, uaddr2 emulator.Pointer, val3 uint32) int64 {
	const (
		FUTEX_WAIT = iota
		FUTEX_WAKE
		FUTEX_FD
		FUTEX_REQUEUE
		FUTEX_CMP_REQUEUE
		FUTEX_WAKE_OP
		FUTEX_LOCK_PI
		FUTEX_UNLOCK_PI
		FUTEX_TRYLOCK_PI
		FUTEX_WAIT_BITSET
		FUTEX_WAKE_BITSET
		FUTEX_WAIT_REQUEUE_PI
		FUTEX_CMP_REQUEUE_PI
		FUTEX_PRIVATE_FLAG   = 128
		FUTEX_CLOCK_REALTIME = 256
		FUTEX_CMD_MASK       = ^(FUTEX_PRIVATE_FLAG | FUTEX_CLOCK_REALTIME)
	)

	switch op & FUTEX_CMD_MASK {
	case FUTEX_WAIT:
		var raw uint32
		err := uaddr.MemReadPtr(4, unsafe.Pointer(&raw))
		if err != nil {
			ctx.SetErrno(linux.EFAULT)
			return -1
		}
		if raw != val {
			ctx.SetErrno(linux.EAGAIN)
			return -1
		}
		var timeout <-chan time.Time
		if !utime.IsNil() {
			var ts timespec
			err = utime.MemReadPtr(uint64(unsafe.Sizeof(ts)), unsafe.Pointer(&ts))
			if err != nil {
				ctx.SetErrno(linux.EFAULT)
				return -1
			}
			d := time.Duration(ts.tv_sec)*time.Second + time.Duration(ts.tv_nsec)
			timeout = time.After(d)
		}
		ch := f.addAwait(uaddr)
		defer f.delAwait(uaddr)
		for {
			select {
			case <-timeout:
				ctx.SetErrno(linux.ETIMEDOUT)
				return -1
			case raw, ok := <-ch:
				if !ok {
					ctx.SetErrno(linux.EPERM)
					return -1
				}
				if raw != val {
					return 0
				}
			}
		}
	case FUTEX_WAKE:
		ch := f.getAwait(uaddr)
		if ch == nil {
			ctx.SetErrno(linux.EAGAIN)
			return -1
		}
		var value uint32
		err := uaddr.MemReadPtr(4, unsafe.Pointer(&value))
		if err != nil {
			ctx.SetErrno(linux.EFAULT)
			return -1
		}
		count := int64(val)
		for i := int64(0); i != count; i++ {
			select {
			case ch <- value:
			default:
				return i
			}
		}
		return count
	case FUTEX_CMP_REQUEUE:
		panic(fmt.Sprint("futex: FUTEX_CMP_REQUEUE", uaddr, val, uaddr2, val3))
	case FUTEX_WAIT_BITSET:
		var raw uint32
		err := uaddr.MemReadPtr(4, unsafe.Pointer(&raw))
		if err != nil {
			ctx.SetErrno(linux.EFAULT)
			return -1
		}
		if raw != val {
			ctx.SetErrno(linux.EAGAIN)
			return -1
		}
		var timeout <-chan time.Time
		if !utime.IsNil() {
			var ts timespec
			err = utime.MemReadPtr(uint64(unsafe.Sizeof(ts)), unsafe.Pointer(&ts))
			if err != nil {
				ctx.SetErrno(linux.EFAULT)
				return -1
			}
			d := time.Duration(ts.tv_sec)*time.Second + time.Duration(ts.tv_nsec)
			timeout = time.After(d)
		}
		ch1, ch2 := f.addBitAwait(uaddr, val3)
		defer f.delBitAwait(uaddr, val3)
		for {
			select {
			case <-timeout:
				ctx.SetErrno(linux.ETIMEDOUT)
				return -1
			case raw, ok := <-ch1:
				if !ok {
					ctx.SetErrno(linux.EPERM)
					return -1
				}
				if raw != val {
					return 0
				}
			case <-ch2:
				return 0
			}
		}
	case FUTEX_WAKE_BITSET:
		count := int64(val)
		i := int64(0)
		f.rw.RLock()
		defer f.rw.RUnlock()
		for bit, await := range f.bitAwaits {
			if bit&val3 == 0 {
				continue
			}
			close(await.ch)
			i += int64(await.ref)
			if i >= count {
				break
			}
		}
		return i
	}
	// panic(fmt.Errorf("futex: %d %w", op, errors.ErrUnsupported))
	ctx.SetErrno(linux.ENOSYS)
	return -1
}

func (f *futex) getAwait(addr emulator.Pointer) chan<- uint32 {
	f.rw.RLock()
	defer f.rw.RUnlock()
	if await, ok := f.awaits[addr.Address()]; ok {
		return await.ch
	}
	return nil
}

func (f *futex) addAwait(addr emulator.Pointer) <-chan uint32 {
	ptr := addr.Address()
	f.rw.Lock()
	defer f.rw.Unlock()
	if await, ok := f.awaits[ptr]; ok {
		await.ref++
		return await.ch
	}
	await := &futexAwait{ch: make(chan uint32), ref: 1}
	f.awaits[ptr] = await
	return await.ch
}

func (f *futex) delAwait(addr emulator.Pointer) {
	ptr := addr.Address()
	f.rw.Lock()
	defer f.rw.Unlock()
	if await, ok := f.awaits[ptr]; ok {
		await.ref--
		if await.ref <= 0 {
			delete(f.awaits, ptr)
		}
	}
}

func (f *futex) addBitAwait(addr emulator.Pointer, bit uint32) (<-chan uint32, <-chan struct{}) {
	ptr := addr.Address()
	f.rw.Lock()
	defer f.rw.Unlock()
	var ch1 <-chan uint32
	if await, ok := f.awaits[ptr]; ok {
		await.ref++
		ch1 = await.ch
	} else {
		await := &futexAwait{ch: make(chan uint32), ref: 1}
		f.awaits[ptr] = await
		ch1 = await.ch
	}
	if await, ok := f.bitAwaits[bit]; ok {
		await.ref++
		return ch1, await.ch
	}
	await := &futexBitAwait{ch: make(chan struct{}), ref: 1}
	f.bitAwaits[bit] = await
	return ch1, await.ch
}

func (f *futex) delBitAwait(addr emulator.Pointer, bit uint32) {
	ptr := addr.Address()
	f.rw.Lock()
	defer f.rw.Unlock()
	if await, ok := f.awaits[ptr]; ok {
		await.ref--
		if await.ref <= 0 {
			delete(f.awaits, ptr)
		}
	}
	if await, ok := f.bitAwaits[bit]; ok {
		await.ref--
		if await.ref <= 0 {
			delete(f.bitAwaits, bit)
		}
	}
}
