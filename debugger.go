package linux

import (
	"github.com/wnxd/microdbg/debugger"
)

type Debugger interface {
	debugger.Debugger
	Kernel
}
