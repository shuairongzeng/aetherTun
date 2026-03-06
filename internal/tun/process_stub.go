//go:build !windows

package tun

// Non-Windows stubs: process name lookup is unsupported.
func lookupTCPProcess(_ [4]byte, _ uint16) string { return "" }
func lookupUDPProcess(_ [4]byte, _ uint16) string { return "" }
func detectProxyProcessName(_ uint16) string      { return "" }
