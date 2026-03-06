//go:build windows

package tun

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const ipUnicastIF = 31 // IP_UNICAST_IF socket option

// getDefaultInterfaceIndex returns the OS interface index used for public
// traffic. Must be called BEFORE TUN routes override the default route.
func getDefaultInterfaceIndex() uint32 {
	proc := modIphlpapi.NewProc("GetBestInterface")
	var dst uint32 = 0x01010101 // 1.1.1.1
	var ifIdx uint32
	ret, _, _ := proc.Call(uintptr(dst), uintptr(unsafe.Pointer(&ifIdx)))
	if ret != 0 {
		return 0
	}
	return ifIdx
}

// protectSocket sets IP_UNICAST_IF on the socket so outbound traffic from
// aether.exe bypasses the TUN adapter and goes through the physical NIC.
func (e *Engine) protectSocket(network, address string, c syscall.RawConn) error {
	if e.defaultIfIndex == 0 {
		return nil
	}
	var sockErr error
	if err := c.Control(func(fd uintptr) {
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], e.defaultIfIndex)
		nbo := *(*uint32)(unsafe.Pointer(&buf[0]))
		sockErr = syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IP, ipUnicastIF, int(nbo))
	}); err != nil {
		return err
	}
	return sockErr
}

// configureWindowsAdapter 為 wintun 適配器分配 IP，並（若啟用）配置系統路由和 DNS
func (e *Engine) configureWindowsAdapter() error {
	prefix, err := netip.ParsePrefix(e.cfg.Tun.Address)
	if err != nil {
		return fmt.Errorf("解析 TUN 地址失敗: %w", err)
	}

	name := e.cfg.Tun.AdapterName
	tunIP := prefix.Addr().String()
	ones := prefix.Bits()
	mask := net.CIDRMask(ones, 32)
	maskStr := fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])

	// 等適配器在系統裡出現（最多 3 秒）
	if err := waitAdapterReady(name, 3*time.Second); err != nil {
		return err
	}

	// 設置主 IP 地址
	if out, err := run("netsh", "interface", "ip", "set", "address",
		"name="+name, "source=static",
		"address="+tunIP, "mask="+maskStr); err != nil {
		return fmt.Errorf("配置 TUN IP 失敗: %v\n%s", err, out)
	}

	// DNS IP（198.18.0.2）作為輔助地址加到適配器上，這樣 DNS server 才能 bind
	dnsHost, _, _ := net.SplitHostPort(e.cfg.Tun.DNSListen)
	if dnsHost != "" && dnsHost != tunIP {
		// 忽略錯誤（可能已存在）
		run("netsh", "interface", "ip", "add", "address", //nolint
			"name="+name, "address="+dnsHost, "mask="+maskStr)
	}

	if !e.cfg.Tun.AutoRoute {
		return nil
	}

	// 設置默認路由（metric=1 優先級最高）
	run("netsh", "interface", "ip", "add", "route", //nolint
		"0.0.0.0/0", name, tunIP, "metric=1", "store=active")

	// 設置 TUN 適配器 DNS 為 FakeIP DNS 地址
	if dnsHost != "" {
		run("netsh", "interface", "ip", "set", "dnsservers", //nolint
			"name="+name, "source=static", "address="+dnsHost, "register=none")
	}

	return nil
}

// teardownWindowsAdapter 清理路由和 IP（Stop 時調用）
func (e *Engine) teardownWindowsAdapter() {
	name := e.cfg.Tun.AdapterName
	if e.cfg.Tun.AutoRoute {
		run("netsh", "interface", "ip", "delete", "route", "0.0.0.0/0", name) //nolint
		run("netsh", "interface", "ip", "set", "dnsservers",                   //nolint
			"name="+name, "source=dhcp")
	}
}

// waitAdapterReady 等待適配器出現在系統介面列表中
func waitAdapterReady(name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, _ := run("netsh", "interface", "show", "interface", name)
		if strings.Contains(out, name) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("等待適配器 %q 超時", name)
}

func run(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}
