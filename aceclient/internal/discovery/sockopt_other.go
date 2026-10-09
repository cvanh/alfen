//go:build !linux

package discovery

import "net"

// restrictToJoinedInterface is a no-op outside Linux: BSD/macOS deliver
// multicast only to sockets that joined the group on the receiving interface
// (Windows is assumed to behave like the .NET sockets Tmds.MDns used there).
func restrictToJoinedInterface(*net.UDPConn, bool) {}
