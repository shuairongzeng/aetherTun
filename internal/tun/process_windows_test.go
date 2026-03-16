//go:build windows

package tun

import "testing"

func hostToNetworkPort32(port uint16) uint32 {
	return uint32(port>>8) | uint32(port&0x00ff)<<8
}

func TestFindUDP4OwnerPIDPrefersExactMatchOverWildcard(t *testing.T) {
	rows := []udpRow4{
		{LocalAddr: 0, LocalPort: hostToNetworkPort32(10808), OwningPID: 11},
		{LocalAddr: 0x0100007f, LocalPort: hostToNetworkPort32(10808), OwningPID: 22},
	}

	pid, ok := findUDP4OwnerPID(rows, [4]byte{127, 0, 0, 1}, 10808)
	if !ok {
		t.Fatal("expected UDP owner PID match, got none")
	}
	if pid != 22 {
		t.Fatalf("expected exact IPv4 match PID 22, got %d", pid)
	}
}

func TestFindUDP4OwnerPIDFallsBackToWildcardBind(t *testing.T) {
	rows := []udpRow4{
		{LocalAddr: 0, LocalPort: hostToNetworkPort32(10808), OwningPID: 33},
	}

	pid, ok := findUDP4OwnerPID(rows, [4]byte{127, 0, 0, 1}, 10808)
	if !ok {
		t.Fatal("expected wildcard UDP owner PID match, got none")
	}
	if pid != 33 {
		t.Fatalf("expected wildcard IPv4 PID 33, got %d", pid)
	}
}

func TestFindUDP6OwnerPIDFallsBackToWildcardBind(t *testing.T) {
	rows := []udpRow6{
		{LocalPort: hostToNetworkPort32(10808), OwningPID: 44},
	}

	pid, ok := findUDP6OwnerPID(rows, [4]byte{192, 168, 44, 105}, 10808)
	if !ok {
		t.Fatal("expected IPv6 wildcard UDP owner PID match, got none")
	}
	if pid != 44 {
		t.Fatalf("expected IPv6 wildcard PID 44, got %d", pid)
	}
}

func TestFindUDP6OwnerPIDMatchesIPv4MappedAddress(t *testing.T) {
	var mapped [16]byte
	mapped[10] = 0xff
	mapped[11] = 0xff
	mapped[12] = 192
	mapped[13] = 168
	mapped[14] = 44
	mapped[15] = 105

	rows := []udpRow6{
		{LocalAddr: mapped, LocalPort: hostToNetworkPort32(10808), OwningPID: 55},
	}

	pid, ok := findUDP6OwnerPID(rows, [4]byte{192, 168, 44, 105}, 10808)
	if !ok {
		t.Fatal("expected IPv4-mapped IPv6 UDP owner PID match, got none")
	}
	if pid != 55 {
		t.Fatalf("expected IPv4-mapped IPv6 PID 55, got %d", pid)
	}
}
