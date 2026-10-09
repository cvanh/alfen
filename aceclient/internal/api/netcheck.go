package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// pingTimeout is the 500 ms Ping.SendPingAsync timeout of CheckNetworkAndPing.
const pingTimeout = 500 * time.Millisecond

// CheckNetworkAndPing ports ICULanDevice.CheckNetworkAndPing
// (ACENetwork/ICUNetwork/ICULanDevice.cs). It returns (true, "") when the
// charger answers, otherwise the exact diagnostic text the installer shows:
//
//   - no usable interface: "No active network available, please make sure you are connected."
//   - charger silent: "Failed to communicate with the charger. ... Available networks: ..."
//     listing up to ten IPv4 addresses of Ethernet-like interfaces as
//     zero-padded dotted quads ("192.168.001.010\t255.255.255.000\teth0");
//   - lookup failure: "Charger with IP address <ip> is not reachable."
//
// Deviations (stdlib has no portable unprivileged ICMP): the echo request is
// replaced by a 500 ms TCP connect to ip:port (443 when port is 0), where an
// accepted or actively refused connection counts as reachable; "Ethernet"
// interfaces are approximated as non-loopback, non point-to-point interfaces
// with a 6-byte hardware address (Go cannot tell Ethernet from Wi-Fi).
func CheckNetworkAndPing(ctx context.Context, ip string, port int) (bool, string) {
	if ctx == nil {
		ctx = context.Background()
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return false, fmt.Sprintf("Charger with IP address %s is not reachable.", ip)
	}
	if !networkAvailable(ifaces) {
		return false, "No active network available, please make sure you are connected."
	}
	if reachable(ctx, ip, port, pingTimeout) {
		return true, ""
	}
	details := strings.Join(ethernetIPv4Lines(ifaces, 10), "\n")
	local, err := localIPAddress(ctx)
	if err != nil {
		return false, fmt.Sprintf("Charger with IP address %s is not reachable.", ip)
	}
	return false, "Failed to communicate with the charger.\n\n" +
		"Charger IP address: " + ip + "\n\n" +
		"Your local IP address: " + local +
		"\n\nIs the charger on the right subnet?\n\nAvailable networks:\n" + details
}

// networkAvailable ports NetworkInterface.GetIsNetworkAvailable: any
// interface that is up and is neither loopback nor a tunnel.
func networkAvailable(ifaces []net.Interface) bool {
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp != 0 && ifc.Flags&net.FlagLoopback == 0 && ifc.Flags&net.FlagPointToPoint == 0 {
			return true
		}
	}
	return false
}

// ethernetIPv4Lines ports the "Available networks" LINQ query: IPv4 unicast
// addresses of Ethernet interfaces, "address\tmask\tname", Take(max).
func ethernetIPv4Lines(ifaces []net.Interface, max int) []string {
	var out []string
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagLoopback != 0 || ifc.Flags&net.FlagPointToPoint != 0 || len(ifc.HardwareAddr) != 6 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok || ipn.IP.To4() == nil {
				continue
			}
			mask := ipn.Mask
			if len(mask) == net.IPv6len {
				mask = mask[12:]
			}
			out = append(out, formatIPAddress(ipn.IP.To4())+"\t"+formatIPAddress(net.IP(mask))+"\t"+ifc.Name)
			if len(out) == max {
				return out
			}
		}
	}
	return out
}

// formatIPAddress ports ICULanDevice.FormatIpAddress: every byte as "D3".
func formatIPAddress(ip net.IP) string {
	parts := make([]string, len(ip))
	for i, b := range ip {
		parts[i] = fmt.Sprintf("%03d", b)
	}
	return strings.Join(parts, ".")
}

// localIPAddress ports ICULanDevice.GetLocalIpAddress: the local address of
// an IPv4 UDP socket "connected" to 8.8.8.8:65530 (no packet is sent).
func localIPAddress(ctx context.Context) (string, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "udp4", "8.8.8.8:65530")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	ua, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", errors.New("no local address")
	}
	return ua.IP.String(), nil
}

// reachable is the echo probe CheckNetworkAndPing uses (tests replace it).
var reachable = probeReachable

// probeReachable stands in for the ICMP echo: a TCP connect within timeout
// that succeeds or is actively refused means the host answered.
func probeReachable(ctx context.Context, ip string, port int, timeout time.Duration) bool {
	if port == 0 {
		port = 443
	}
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
	if err == nil {
		conn.Close()
		return true
	}
	return isConnRefused(err)
}

// isConnRefused reports ECONNREFUSED (WSAECONNREFUSED = 10061 on Windows).
func isConnRefused(err error) bool {
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	var errno syscall.Errno
	return errors.As(err, &errno) && errno == 10061
}
