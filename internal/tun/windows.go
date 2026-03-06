//go:build windows

package tun

import (
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

// getPhysicalInterfaceIP 返回指定网卡索引的首个 IPv4 单播地址。
// 在 TUN 接管默认路由前调用，用于 protectSocket 绑定物理网卡。
func getPhysicalInterfaceIP(ifIndex uint32) net.IP {
	iface, err := net.InterfaceByIndex(int(ifIndex))
	if err != nil {
		return nil
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip4 := ip.To4(); ip4 != nil && !ip4.IsLoopback() {
			return ip4
		}
	}
	return nil
}

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

// protectSocket 将 socket 绑定到物理网卡 IP，使出站连接绕过 TUN 默认路由。
// Windows 强主机模型保证：源 IP 属于哪个接口，报文就从那个接口出去。
func (e *Engine) protectSocket(network, address string, c syscall.RawConn) error {
	if e.localIP == nil {
		return nil
	}
	ip4 := e.localIP.To4()
	if ip4 == nil {
		return nil
	}
	var innerErr error
	c.Control(func(fd uintptr) {
		sa := &syscall.SockaddrInet4{}
		copy(sa.Addr[:], ip4)
		innerErr = syscall.Bind(syscall.Handle(fd), sa)
	})
	return innerErr
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
