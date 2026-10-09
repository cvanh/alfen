//go:build linux

package discovery

import (
	"net"
	"syscall"
)

// Linux socket options (linux/in.h, linux/in6.h); not exported by package
// syscall on every architecture.
const (
	ipMulticastAll   = 49 // IP_MULTICAST_ALL
	ipv6MulticastAll = 29 // IPV6_MULTICAST_ALL (Linux >= 4.20)
)

// restrictToJoinedInterface clears IP(V6)_MULTICAST_ALL so the per-interface
// socket only receives the group traffic of the interface it joined. Linux
// otherwise delivers 224.0.0.251 traffic from every interface to every socket
// bound to :5353, while Tmds.MDns relies on each NetworkInterfaceHandler's
// socket seeing only its own interface (the Windows/BSD behaviour). Failures
// (older kernels) are ignored.
func restrictToJoinedInterface(c *net.UDPConn, v6 bool) {
	rc, err := c.SyscallConn()
	if err != nil {
		return
	}
	_ = rc.Control(func(fd uintptr) {
		if v6 {
			_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, ipv6MulticastAll, 0)
		} else {
			_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, ipMulticastAll, 0)
		}
	})
}
