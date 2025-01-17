package arm

import (
	"github.com/wnxd/microdbg-linux/gcc"
	"github.com/wnxd/microdbg/debugger"
)

const POINTER_SIZE = 4

func NewVaList(dbg debugger.Debugger, ptr uint64) (debugger.Args, error) {
	return gcc.NewStdVaList(dbg, ptr, POINTER_SIZE)
}
