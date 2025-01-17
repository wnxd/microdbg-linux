package arm64

import (
	"errors"
	"unsafe"

	"github.com/wnxd/microdbg-linux/internal"
	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
	"github.com/wnxd/microdbg/encoding"
)

const POINTER_SIZE = 8

type list struct {
	emu       emulator.Emulator `encoding:"ignore"`
	__stack   uint64
	__gr_top  uint64
	__vr_top  uint64
	__gr_offs int32
	__vr_offs int32
}

func NewVaList(dbg debugger.Debugger, ptr uint64) (debugger.Args, error) {
	var list list
	err := dbg.MemExtract(ptr, &list)
	if err != nil {
		return nil, err
	}
	list.emu = dbg.Emulator()
	return &list, nil
}

func (va *list) Extract(args ...any) error {
	for _, arg := range args {
		err := encoding.Decode(va, arg)
		if err != nil {
			return err
		}
		va.__stack = debugger.Align(va.__stack, POINTER_SIZE)
		va.__gr_offs = debugger.Align(va.__gr_offs, POINTER_SIZE)
	}
	return nil
}

func (va *list) BlockSize() int {
	return 8
}

func (va *list) Offset() uint64 {
	return 0
}

func (va *list) Skip(n int) error {
	if va.__gr_offs < 0 {
		va.__gr_offs += int32(n)
	} else {
		va.__stack += uint64(n)
	}
	return nil
}

func (va *list) Read(b []byte) (n int, err error) {
	if n = len(b); va.__gr_offs >= 0 {
		err = va.emu.MemReadPtr(va.__stack, uint64(n), unsafe.Pointer(unsafe.SliceData(b)))
		va.__stack += uint64(n)
	} else if x := n + int(va.__gr_offs); x <= 0 {
		err = va.emu.MemReadPtr(va.__gr_top+uint64(va.__gr_offs), uint64(n), unsafe.Pointer(unsafe.SliceData(b)))
		va.__gr_offs += int32(n)
	} else {
		i := n - x
		err = va.emu.MemReadPtr(va.__gr_top+uint64(va.__gr_offs), uint64(i), unsafe.Pointer(unsafe.SliceData(b)))
		if err != nil {
			return
		}
		va.__gr_offs = 0
		err = va.emu.MemReadPtr(va.__stack, uint64(x), unsafe.Pointer(unsafe.SliceData(b[i:])))
		va.__stack += uint64(x)
	}
	return
}

func (va *list) ReadFloat() (float32, error) {
	d, err := va.ReadDouble()
	if err != nil {
		return 0, err
	}
	return float32(d), nil
}

func (va *list) ReadDouble() (float64, error) {
	var d float64
	if va.__vr_offs >= 0 {
		err := va.emu.MemReadPtr(va.__stack, 8, unsafe.Pointer(&d))
		va.__stack += 8
		return d, err
	}
	err := va.emu.MemReadPtr(va.__vr_top+uint64(va.__vr_offs), 8, unsafe.Pointer(&d))
	va.__vr_offs += 16
	return d, err
}

func (va *list) ReadString() (string, error) {
	return "", errors.ErrUnsupported
}

func (va *list) ReadStream() (encoding.Stream, error) {
	var addr uint64
	_, err := va.Read(unsafe.Slice((*byte)(unsafe.Pointer(&addr)), 8))
	if err != nil {
		return nil, err
	}
	return internal.PointerStream(emulator.ToPointer(va.emu, addr), nil, POINTER_SIZE), nil
}

func (va *list) Write([]byte) (int, error) {
	return 0, errors.ErrUnsupported
}

func (va *list) WriteFloat(float32) error {
	return errors.ErrUnsupported
}

func (va *list) WriteDouble(float64) error {
	return errors.ErrUnsupported
}

func (va *list) WriteString(string) error {
	return errors.ErrUnsupported
}

func (va *list) WriteStream(int) (encoding.Stream, error) {
	return nil, errors.ErrUnsupported
}
