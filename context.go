package linux

import (
	"github.com/wnxd/microdbg/debugger"
)

type Context interface {
	debugger.Context
	Errno() Errno
	SetErrno(err Errno)
}

type context struct {
	debugger.Context
	dbg Debugger
}

func NewContext(ctx debugger.Context, dbg Debugger) Context {
	return &context{ctx, dbg}
}

func (ctx *context) Errno() Errno {
	return ctx.dbg.Errno()
}

func (ctx *context) SetErrno(err Errno) {
	ctx.dbg.SetErrno(err)
}
