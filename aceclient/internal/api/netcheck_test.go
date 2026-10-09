package api

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestFormatIPAddress(t *testing.T) {
	if got := formatIPAddress(net.IPv4(192, 168, 1, 10).To4()); got != "192.168.001.010" {
		t.Fatalf("got %q", got)
	}
	if got := formatIPAddress(net.IP(net.CIDRMask(24, 32))); got != "255.255.255.000" {
		t.Fatalf("mask %q", got)
	}
}

func TestProbeReachable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("cannot listen:", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if !probeReachable(context.Background(), "127.0.0.1", port, pingTimeout) {
		t.Fatal("listening port must be reachable")
	}
	ln.Close()
	// An actively refused connection still proves the host is there.
	if !probeReachable(context.Background(), "127.0.0.1", port, pingTimeout) {
		t.Fatal("refused connection must count as reachable")
	}
}

func TestCheckNetworkAndPing(t *testing.T) {
	ifaces, err := net.Interfaces()
	if err != nil || !networkAvailable(ifaces) {
		t.Skip("no usable network interface")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("cannot listen:", err)
	}
	defer ln.Close()
	if ok, msg := CheckNetworkAndPing(context.Background(), "127.0.0.1", ln.Addr().(*net.TCPAddr).Port); !ok || msg != "" {
		t.Fatalf("reachable host: %v %q", ok, msg)
	}
	// A silent charger: the message is one of the two C# texts.
	reachable = func(context.Context, string, int, time.Duration) bool { return false }
	defer func() { reachable = probeReachable }()
	ok, msg := CheckNetworkAndPing(context.Background(), "192.0.2.1", 443)
	if ok {
		t.Fatal("probe failed but reported reachable")
	}
	if !strings.HasPrefix(msg, "Failed to communicate with the charger.\n\nCharger IP address: 192.0.2.1\n\nYour local IP address: ") &&
		msg != "Charger with IP address 192.0.2.1 is not reachable." {
		t.Fatalf("message %q", msg)
	}
	if strings.HasPrefix(msg, "Failed") && !strings.Contains(msg, "\n\nIs the charger on the right subnet?\n\nAvailable networks:\n") {
		t.Fatalf("message %q", msg)
	}
}
