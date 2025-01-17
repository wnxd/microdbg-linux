package x86_64

import (
	"errors"
	"unsafe"

	"github.com/wnxd/microdbg-linux/internal"
	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
	"github.com/wnxd/microdbg/encoding"
)

const (
	POINTER_SIZE       = 8
	GENERAL_OFFSET_MAX = 6 * POINTER_SIZE
	XMM_OFFSET_MAX     = GENERAL_OFFSET_MAX + 8*16
)

type list struct {
	emu               emulator.Emulator `encoding:"ignore"`
	gp_offset         uint32
	fp_offset         uint32
	overflow_arg_area uint64
	reg_save_area     uint64
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
		va.gp_offset = debugger.Align(va.gp_offset, POINTER_SIZE)
		va.overflow_arg_area = debugger.Align(va.overflow_arg_area, uint64(POINTER_SIZE))
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
	if va.gp_offset < GENERAL_OFFSET_MAX {
		va.gp_offset += uint32(n)
	} else {
		va.overflow_arg_area += uint64(n)
	}
	return nil
}

func (va *list) Read(b []byte) (n int, err error) {
	if n = len(b); va.gp_offset >= GENERAL_OFFSET_MAX {
		err = va.emu.MemReadPtr(va.overflow_arg_area, uint64(n), unsafe.Pointer(unsafe.SliceData(b)))
		va.overflow_arg_area += uint64(n)
	} else if n+int(va.gp_offset) <= GENERAL_OFFSET_MAX {
		err = va.emu.MemReadPtr(va.reg_save_area+uint64(va.gp_offset), uint64(n), unsafe.Pointer(unsafe.SliceData(b)))
		va.gp_offset += uint32(n)
	} else {
		i := uint64(GENERAL_OFFSET_MAX - va.gp_offset)
		err = va.emu.MemReadPtr(va.reg_save_area+uint64(va.gp_offset), i, unsafe.Pointer(unsafe.SliceData(b)))
		if err != nil {
			return
		}
		va.gp_offset = GENERAL_OFFSET_MAX
		x := uint64(n) - i
		err = va.emu.MemReadPtr(va.overflow_arg_area, x, unsafe.Pointer(unsafe.SliceData(b[i:])))
		va.overflow_arg_area += x
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
	if va.fp_offset >= XMM_OFFSET_MAX {
		err := va.emu.MemReadPtr(va.overflow_arg_area, 8, unsafe.Pointer(&d))
		va.overflow_arg_area += 8
		return d, err
	}
	err := va.emu.MemReadPtr(va.reg_save_area+uint64(va.fp_offset), 8, unsafe.Pointer(&d))
	va.fp_offset += 16
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
