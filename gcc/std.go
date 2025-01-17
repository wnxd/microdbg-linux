package gcc

import (
	"errors"
	"unsafe"

	"github.com/wnxd/microdbg-linux/internal"
	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
	"github.com/wnxd/microdbg/encoding"
)

type list struct {
	emu  emulator.Emulator
	ptr  uint64
	size int
}

func NewStdVaList(dbg debugger.Debugger, ptr, size uint64) (debugger.Args, error) {
	if ptr == 0 || size == 0 {
		return nil, debugger.ErrArgumentInvalid
	}
	return &list{emu: dbg.Emulator(), ptr: ptr, size: int(size)}, nil
}

func (va *list) Extract(args ...any) error {
	for _, arg := range args {
		err := encoding.Decode(va, arg)
		if err != nil {
			return err
		}
		va.ptr = debugger.Align(va.ptr, 4)
	}
	return nil
}

func (va *list) BlockSize() int {
	return va.size
}

func (va *list) Offset() uint64 {
	return 0
}

func (va *list) Skip(n int) error {
	va.ptr += uint64(n)
	return nil
}

func (va *list) Read(b []byte) (int, error) {
	n := len(b)
	err := va.emu.MemReadPtr(va.ptr, uint64(n), unsafe.Pointer(unsafe.SliceData(b)))
	va.ptr += uint64(n)
	return n, err
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
	_, err := va.Read(unsafe.Slice((*byte)(unsafe.Pointer(&d)), 8))
	return d, err
}

func (va *list) ReadString() (string, error) {
	return "", errors.ErrUnsupported
}

func (va *list) ReadStream() (encoding.Stream, error) {
	var addr uint64
	_, err := va.Read(unsafe.Slice((*byte)(unsafe.Pointer(&addr)), va.size))
	if err != nil {
		return nil, err
	}
	return internal.PointerStream(emulator.ToPointer(va.emu, addr), nil, va.size), nil
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
