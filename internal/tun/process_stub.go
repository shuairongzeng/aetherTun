//go:build !windows

package tun

import (
	"net"
	"syscall"
)

func lookupTCPProcess(_ [4]byte, _ uint16) string { return "" }
func lookupUDPProcess(_ [4]byte, _ uint16) string { return "" }
func detectProxyProcessName(_ uint16) string      { return "" }
func getDefaultInterfaceIndex() uint32            { return 0 }
func getDefaultGateway(_ uint32) (string, string, error) {
	return "", "", nil
}
func getPhysicalInterfaceIP(_ uint32) net.IP { return nil }

func (e *Engine) protectSocket(_, _ string, _ syscall.RawConn) error { return nil }
func (e *Engine) runDirectSocketDiagnostics(_, _ string)             {}
func (e *Engine) acquireDirectRoute(_ string) (func(), error)        { return func() {}, nil }
