//go:build !windows

package tun

import "syscall"

func lookupTCPProcess(_ [4]byte, _ uint16) string { return "" }
func lookupUDPProcess(_ [4]byte, _ uint16) string { return "" }
func detectProxyProcessName(_ uint16) string      { return "" }
func getDefaultInterfaceIndex() uint32             { return 0 }

func (e *Engine) protectSocket(_, _ string, _ syscall.RawConn) error { return nil }
