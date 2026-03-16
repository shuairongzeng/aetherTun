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
	afINET6             = 23
)

type tcpRow4 struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
	OwningPID  uint32
}

type udpRow4 struct {
	LocalAddr uint32
	LocalPort uint32
	OwningPID uint32
}

type udpRow6 struct {
	LocalAddr    [16]byte
	LocalScopeID uint32
	LocalPort    uint32
	OwningPID    uint32
}

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

func lookupUDPProcess(srcIP [4]byte, srcPort uint16) string {
	if pid, ok := findUDP4OwnerPID(loadUDP4Rows(), srcIP, srcPort); ok {
		return pidToName(pid)
	}
	if pid, ok := findUDP6OwnerPID(loadUDP6Rows(), srcIP, srcPort); ok {
		return pidToName(pid)
	}
	return ""
}

func loadUDP4Rows() []udpRow4 {
	buf := getTable(procGetExtendedUdpTable, afINET, udpTableOwnerPID)
	if len(buf) < 4 {
		return nil
	}
	n := *(*uint32)(unsafe.Pointer(&buf[0]))
	rows := make([]udpRow4, 0, n)
	const rowSz = unsafe.Sizeof(udpRow4{})
	base := uintptr(unsafe.Pointer(&buf[4]))
	for i := uint32(0); i < n; i++ {
		row := (*udpRow4)(unsafe.Pointer(base + uintptr(i)*rowSz))
		rows = append(rows, *row)
	}
	return rows
}

func loadUDP6Rows() []udpRow6 {
	buf := getTable(procGetExtendedUdpTable, afINET6, udpTableOwnerPID)
	if len(buf) < 4 {
		return nil
	}
	n := *(*uint32)(unsafe.Pointer(&buf[0]))
	rows := make([]udpRow6, 0, n)
	const rowSz = unsafe.Sizeof(udpRow6{})
	base := uintptr(unsafe.Pointer(&buf[4]))
	for i := uint32(0); i < n; i++ {
		row := (*udpRow6)(unsafe.Pointer(base + uintptr(i)*rowSz))
		rows = append(rows, *row)
	}
	return rows
}

func findUDP4OwnerPID(rows []udpRow4, srcIP [4]byte, srcPort uint16) (uint32, bool) {
	var wildcardPID uint32
	for _, row := range rows {
		rowPort := ntohsU32(row.LocalPort)
		if rowPort != srcPort {
			continue
		}

		rowIP := *(*[4]byte)(unsafe.Pointer(&row.LocalAddr))
		if rowIP == srcIP {
			return row.OwningPID, true
		}
		if rowIP == ([4]byte{}) && wildcardPID == 0 {
			wildcardPID = row.OwningPID
		}
	}
	if wildcardPID != 0 {
		return wildcardPID, true
	}
	return 0, false
}

func findUDP6OwnerPID(rows []udpRow6, srcIP [4]byte, srcPort uint16) (uint32, bool) {
	var wildcardPID uint32
	mappedIP := ipv4MappedIPv6(srcIP)
	for _, row := range rows {
		rowPort := ntohsU32(row.LocalPort)
		if rowPort != srcPort {
			continue
		}

		if row.LocalAddr == mappedIP {
			return row.OwningPID, true
		}
		if row.LocalAddr == ([16]byte{}) && wildcardPID == 0 {
			wildcardPID = row.OwningPID
		}
	}
	if wildcardPID != 0 {
		return wildcardPID, true
	}
	return 0, false
}

func ipv4MappedIPv6(ip [4]byte) [16]byte {
	var mapped [16]byte
	mapped[10] = 0xff
	mapped[11] = 0xff
	copy(mapped[12:], ip[:])
	return mapped
}

func getTable(proc *syscall.LazyProc, af, tableClass uintptr) []byte {
	size := uint32(8192)
	for attempt := 0; attempt < 4; attempt++ {
		buf := make([]byte, size)
		ret, _, _ := proc.Call(
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&size)),
			0,
			af,
			tableClass,
			0,
		)
		if ret == 0 {
			return buf
		}
		if ret != 122 {
			return nil
		}
	}
	return nil
}

func ntohsU32(v uint32) uint16 {
	b := (*[4]byte)(unsafe.Pointer(&v))
	return uint16(b[0])<<8 | uint16(b[1])
}

func makeKey(ip [4]byte, port uint16) uint64 {
	return uint64(ip[0])<<40 | uint64(ip[1])<<32 | uint64(ip[2])<<24 | uint64(ip[3])<<16 | uint64(port)
}

func detectProxyProcessName(proxyPort uint16) string {
	buf := getTable(procGetExtendedTcpTable, afINET, tcpTableOwnerPIDAll)
	if buf == nil {
		return ""
	}
	n := *(*uint32)(unsafe.Pointer(&buf[0]))
	const rowSz = unsafe.Sizeof(tcpRow4{})
	const listenState = 2
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
