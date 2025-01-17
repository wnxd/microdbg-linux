package kernel

import (
	linux "github.com/wnxd/microdbg-linux"
	"github.com/wnxd/microdbg/debugger"
)

type Kernel interface {
	linux.Kernel
	KernelInit(debugger.Debugger) error
	KernelClose() error
}
