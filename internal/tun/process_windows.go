//go:build windows

package tun

import (
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modIphlpapi             = syscall.NewLazyDLL("iphlpapi.dll")
	procGetExtendedTcpTable = modIphlpapi.NewProc("GetExtendedTcpTable")
	procGetExtendedUdpTable = modIphlpapi.NewProc("GetExtendedUdpTable")
)

const (
	tcpTableOwnerPIDAll = 5
	udpTableOwnerPID    = 1
	afINET              = 2
)

// 对应 MIB_TCPROW_OWNER_PID（6 × uint32 = 24 bytes，无填充）
type tcpRow4 struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
	OwningPID  uint32
}

// 对应 MIB_UDPROW_OWNER_PID（3 × uint32 = 12 bytes，无填充）
type udpRow4 struct {
	LocalAddr uint32
	LocalPort uint32
	OwningPID uint32
}

// lookupTCPProcess 根据连接的源 IP+端口查找所属进程名（小写 exe 文件名）。
// srcIP 和 srcPort 来自 gVisor ForwarderRequest 的 RemoteAddress/RemotePort。
func lookupTCPProcess(srcIP [4]byte, srcPort uint16) string {
	buf := getTable(procGetExtendedTcpTable, afINET, tcpTableOwnerPIDAll)
	if buf == nil {
		return ""
	}
	n := *(*uint32)(unsafe.Pointer(&buf[0]))
	const rowSz = unsafe.Sizeof(tcpRow4{})
	base := uintptr(unsafe.Pointer(&buf[4]))
	want := makeKey(srcIP, srcPort)
	for i := uint32(0); i < n; i++ {
		row := (*tcpRow4)(unsafe.Pointer(base + uintptr(i)*rowSz))
		rowIP := *(*[4]byte)(unsafe.Pointer(&row.LocalAddr))
		rowPort := ntohsU32(row.LocalPort)
		if makeKey(rowIP, rowPort) == want {
			return pidToName(row.OwningPID)
		}
	}
	return ""
}

// lookupUDPProcess 根据源 IP+端口查找 UDP socket 所属进程名。
func lookupUDPProcess(srcIP [4]byte, srcPort uint16) string {
	buf := getTable(procGetExtendedUdpTable, afINET, udpTableOwnerPID)
	if buf == nil {
		return ""
	}
	n := *(*uint32)(unsafe.Pointer(&buf[0]))
	const rowSz = unsafe.Sizeof(udpRow4{})
	base := uintptr(unsafe.Pointer(&buf[4]))
	want := makeKey(srcIP, srcPort)
	for i := uint32(0); i < n; i++ {
		row := (*udpRow4)(unsafe.Pointer(base + uintptr(i)*rowSz))
		rowIP := *(*[4]byte)(unsafe.Pointer(&row.LocalAddr))
		rowPort := ntohsU32(row.LocalPort)
		if makeKey(rowIP, rowPort) == want {
			return pidToName(row.OwningPID)
		}
	}
	return ""
}

// getTable 调用 GetExtended{Tcp|Udp}Table，按需扩大缓冲区，返回原始字节。
func getTable(proc *syscall.LazyProc, af, tableClass uintptr) []byte {
	size := uint32(8192)
	for attempt := 0; attempt < 4; attempt++ {
		buf := make([]byte, size)
		ret, _, _ := proc.Call(
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&size)),
			0, // bOrder = unsorted
			af,
			tableClass,
			0,
		)
		if ret == 0 { // NO_ERROR
			return buf
		}
		if ret != 122 { // not ERROR_INSUFFICIENT_BUFFER
			return nil
		}
		// size 已被更新为所需大小，下一轮用更大的 buf
	}
	return nil
}

// ntohsU32 把 Windows 表里以网络字节序存储的端口（DWORD 低 16 位）转成主机序 uint16。
// 内存布局：[高字节, 低字节, 0x00, 0x00]，Go 以小端 uint32 读取。
func ntohsU32(v uint32) uint16 {
	b := (*[4]byte)(unsafe.Pointer(&v))
	return uint16(b[0])<<8 | uint16(b[1])
}

// makeKey 将 IP+端口打包为 uint64 用于快速比较。
func makeKey(ip [4]byte, port uint16) uint64 {
	return uint64(ip[0])<<40 | uint64(ip[1])<<32 | uint64(ip[2])<<24 | uint64(ip[3])<<16 | uint64(port)
}

// detectProxyProcessName 找出监听 proxyPort 的进程名（小写 exe 文件名）。
// 用于在启动时自动识别代理程序（如 xray.exe），防止其出站流量被 TUN 截获形成环路。
func detectProxyProcessName(proxyPort uint16) string {
	buf := getTable(procGetExtendedTcpTable, afINET, tcpTableOwnerPIDAll)
	if buf == nil {
		return ""
	}
	n := *(*uint32)(unsafe.Pointer(&buf[0]))
	const rowSz = unsafe.Sizeof(tcpRow4{})
	const listenState = 2 // MIB_TCP_STATE_LISTEN
	base := uintptr(unsafe.Pointer(&buf[4]))
	for i := uint32(0); i < n; i++ {
		row := (*tcpRow4)(unsafe.Pointer(base + uintptr(i)*rowSz))
		if row.State == listenState && ntohsU32(row.LocalPort) == proxyPort {
			return pidToName(row.OwningPID)
		}
	}
	return ""
}
func pidToName(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)
	var buf [512]uint16
	n := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		return ""
	}
	return strings.ToLower(filepath.Base(windows.UTF16ToString(buf[:n])))
}
