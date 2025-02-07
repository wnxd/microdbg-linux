package kernel

import (
	"errors"
	"fmt"

	linux "github.com/wnxd/microdbg-linux"
	"github.com/wnxd/microdbg/socket"
)

const (
	AF_UNSPEC     = 0
	AF_UNIX       = 1
	AF_LOCAL      = 1
	AF_INET       = 2
	AF_AX25       = 3
	AF_IPX        = 4
	AF_APPLETALK  = 5
	AF_NETROM     = 6
	AF_BRIDGE     = 7
	AF_ATMPVC     = 8
	AF_X25        = 9
	AF_INET6      = 10
	AF_ROSE       = 11
	AF_DECnet     = 12
	AF_NETBEUI    = 13
	AF_SECURITY   = 14
	AF_KEY        = 15
	AF_NETLINK    = 16
	AF_ROUTE      = AF_NETLINK
	AF_PACKET     = 17
	AF_ASH        = 18
	AF_ECONET     = 19
	AF_ATMSVC     = 20
	AF_RDS        = 21
	AF_SNA        = 22
	AF_IRDA       = 23
	AF_PPPOX      = 24
	AF_WANPIPE    = 25
	AF_LLC        = 26
	AF_CAN        = 29
	AF_TIPC       = 30
	AF_BLUETOOTH  = 31
	AF_IUCV       = 32
	AF_RXRPC      = 33
	AF_ISDN       = 34
	AF_PHONET     = 35
	AF_IEEE802154 = 36
	AF_CAIF       = 37
	AF_ALG        = 38
	AF_NFC        = 39
	AF_VSOCK      = 40
	AF_KCM        = 41
	AF_QIPCRTR    = 42
	AF_MAX        = 43

	SOCK_STREAM    = 1
	SOCK_DGRAM     = 2
	SOCK_RAW       = 3
	SOCK_RDM       = 4
	SOCK_SEQPACKET = 5
	SOCK_DCCP      = 6
	SOCK_PACKET    = 10
)

type network struct {
}

func (n *network) socket(ctx linux.Context, domain, typ, protocol int32) int32 {
	typ &= 0x7ffff
	var network socket.Network
	switch domain {
	case AF_UNSPEC:
		panic(fmt.Errorf("socket: AF_UNSPEC %w", errors.ErrUnsupported))
	case AF_LOCAL:
		switch typ {
		case SOCK_STREAM:
			network = socket.Unix
		case SOCK_DGRAM:
			network = socket.UnixGram
		}
	case AF_INET, AF_INET6:
		switch typ {
		case SOCK_STREAM:
			network = socket.TCP
		case SOCK_DGRAM:
			network = socket.UDP
		}
	}
	if network == "" {
		ctx.SetErrno(linux.EAFNOSUPPORT)
		return -1
	}
	dbg := ctx.Debugger()
	s, err := dbg.NewSocket(network)
	if err != nil {
		ctx.SetErrno(linux.EACCES)
		return -1
	}
	fd := dbg.CreateFileDescriptor(s)
	return int32(fd)
}
