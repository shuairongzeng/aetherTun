//go:build windows

package tun

import (
	"context"
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

func mergeSocketControlError(controlErr, innerErr error) error {
	if controlErr != nil && innerErr == nil {
		return controlErr
	}
	return innerErr
}

func bindSocketToLocalIPv4(c syscall.RawConn, ip4 net.IP) (uintptr, error, error) {
	var fdValue uintptr
	var innerErr error
	controlErr := c.Control(func(fd uintptr) {
		fdValue = fd
		sa := &syscall.SockaddrInet4{}
		copy(sa.Addr[:], ip4)
		innerErr = syscall.Bind(syscall.Handle(fd), sa)
	})
	return fdValue, controlErr, innerErr
}

func (e *Engine) localIPv4() net.IP {
	if e.localIP == nil {
		return nil
	}
	return e.localIP.To4()
}

// protectSocket 将 socket 绑定到物理网卡 IP，使出站连接绕过 TUN 默认路由。
// Windows 强主机模型保证：源 IP 属于哪个接口，报文就从那个接口出去。
func (e *Engine) protectSocket(network, address string, c syscall.RawConn) error {
	if e.localIP == nil {
		// #region agent log
		debugLogCurrent(debugRunInvestigation, "H3", "windows.go:protectSocket", "protectSocket skipped: localIP missing", map[string]interface{}{"network": network, "address": address, "defaultIfIndex": e.defaultIfIndex})
		// #endregion
		return nil
	}
	ip4 := e.localIPv4()
	if ip4 == nil {
		// #region agent log
		debugLogCurrent(debugRunInvestigation, "H3", "windows.go:protectSocket", "protectSocket skipped: localIP not IPv4", map[string]interface{}{"network": network, "address": address, "localIP": debugIPString(e.localIP), "defaultIfIndex": e.defaultIfIndex})
		// #endregion
		return nil
	}
	hypothesisID := "H1"
	if strings.HasPrefix(network, "udp") {
		hypothesisID = "H2"
	}
	fdValue, controlErr, innerErr := bindSocketToLocalIPv4(c, ip4)
	innerErr = mergeSocketControlError(controlErr, innerErr)
	// #region agent log
	debugLogCurrent(debugRunInvestigation, hypothesisID, "windows.go:protectSocket", "protectSocket bind result", map[string]interface{}{"network": network, "address": address, "localIP": debugIPString(ip4), "defaultIfIndex": e.defaultIfIndex, "fd": fdValue, "controlErr": fmt.Sprintf("%v", controlErr), "bindErr": fmt.Sprintf("%v", innerErr)})
	// #endregion
	return innerErr
}

func getDefaultGateway(ifIndex uint32) (string, string, error) {
	ps := fmt.Sprintf("(Get-NetRoute -AddressFamily IPv4 -DestinationPrefix '0.0.0.0/0' -InterfaceIndex %d | Sort-Object RouteMetric, InterfaceMetric | Select-Object -First 1 -ExpandProperty NextHop)", ifIndex)
	out, err := run("powershell", "-NoProfile", "-Command", ps)
	gateway := strings.TrimSpace(out)
	if err != nil {
		return gateway, out, err
	}
	if gateway == "" {
		return "", out, fmt.Errorf("no default gateway found for interface %d", ifIndex)
	}
	return gateway, out, nil
}

func hostOnly(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err == nil {
		return host
	}
	return address
}

func hostRouteHypothesisID(testNetwork, testTarget, failingTarget string) string {
	if strings.EqualFold(testTarget, failingTarget) {
		return "H12"
	}
	if strings.HasPrefix(testNetwork, "udp") {
		return "H11"
	}
	return "H10"
}

func addHostRoute(targetHost, gateway string, ifIndex uint32) (string, error) {
	return run("route", "add", targetHost, "mask", "255.255.255.255", gateway, "metric", "1", "if", fmt.Sprintf("%d", ifIndex))
}

func deleteHostRoute(targetHost, gateway string, ifIndex uint32) (string, error) {
	return run("route", "delete", targetHost, "mask", "255.255.255.255", gateway, "if", fmt.Sprintf("%d", ifIndex))
}

func shouldInstallDirectRoute(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	if ip4.IsLoopback() || ip4.IsPrivate() || ip4.IsLinkLocalUnicast() || ip4.IsMulticast() {
		return false
	}
	return true
}

func changeHostRoute(targetHost, gateway string, ifIndex uint32) (string, error) {
	return run("route", "change", targetHost, "mask", "255.255.255.255", gateway, "metric", "1", "if", fmt.Sprintf("%d", ifIndex))
}

func (e *Engine) acquireDirectRoute(targetHost string) (func(), error) {
	ip := net.ParseIP(targetHost)
	if !shouldInstallDirectRoute(ip) {
		return func() {}, nil
	}
	gateway := e.defaultGateway
	if gateway == "" {
		discoveredGateway, _, err := getDefaultGateway(e.defaultIfIndex)
		if err != nil {
			return nil, err
		}
		gateway = discoveredGateway
		e.defaultGateway = discoveredGateway
	}

	e.routeMu.Lock()
	if refCount := e.routeRefs[targetHost]; refCount > 0 {
		e.routeRefs[targetHost] = refCount + 1
		e.routeMu.Unlock()
		// #region agent log
		debugLogCurrent(debugRunInvestigation, "H13", "windows.go:acquireDirectRoute", "Direct route ref incremented", map[string]interface{}{"targetHost": targetHost, "gateway": gateway, "defaultIfIndex": e.defaultIfIndex, "refCount": refCount + 1})
		// #endregion
		return func() { e.releaseDirectRoute(targetHost) }, nil
	}
	e.routeMu.Unlock()

	routeOp := "add"
	routeOut, routeErr := addHostRoute(targetHost, gateway, e.defaultIfIndex)
	if routeErr != nil {
		routeOp = "change"
		routeOut, routeErr = changeHostRoute(targetHost, gateway, e.defaultIfIndex)
	}
	if routeErr != nil {
		// #region agent log
		debugLogCurrent(debugRunInvestigation, "H13", "windows.go:acquireDirectRoute", "Direct route install failed", map[string]interface{}{"targetHost": targetHost, "gateway": gateway, "defaultIfIndex": e.defaultIfIndex, "routeOp": routeOp, "routeErr": fmt.Sprintf("%v", routeErr), "routeOut": strings.TrimSpace(routeOut)})
		// #endregion
		return nil, routeErr
	}

	e.routeMu.Lock()
	e.routeRefs[targetHost] = 1
	e.routeMu.Unlock()
	// #region agent log
	debugLogCurrent(debugRunInvestigation, "H13", "windows.go:acquireDirectRoute", "Direct route installed", map[string]interface{}{"targetHost": targetHost, "gateway": gateway, "defaultIfIndex": e.defaultIfIndex, "routeOp": routeOp, "routeOut": strings.TrimSpace(routeOut), "refCount": 1})
	// #endregion
	return func() { e.releaseDirectRoute(targetHost) }, nil
}

func (e *Engine) releaseDirectRoute(targetHost string) {
	ip := net.ParseIP(targetHost)
	if !shouldInstallDirectRoute(ip) {
		return
	}

	e.routeMu.Lock()
	refCount := e.routeRefs[targetHost]
	if refCount == 0 {
		e.routeMu.Unlock()
		return
	}
	if refCount > 1 {
		e.routeRefs[targetHost] = refCount - 1
		e.routeMu.Unlock()
		// #region agent log
		debugLogCurrent(debugRunInvestigation, "H13", "windows.go:releaseDirectRoute", "Direct route ref decremented", map[string]interface{}{"targetHost": targetHost, "gateway": e.defaultGateway, "defaultIfIndex": e.defaultIfIndex, "refCount": refCount - 1})
		// #endregion
		return
	}
	delete(e.routeRefs, targetHost)
	gateway := e.defaultGateway
	e.routeMu.Unlock()

	routeOut, routeErr := deleteHostRoute(targetHost, gateway, e.defaultIfIndex)
	// #region agent log
	debugLogCurrent(debugRunInvestigation, "H13", "windows.go:releaseDirectRoute", "Direct route removed", map[string]interface{}{"targetHost": targetHost, "gateway": gateway, "defaultIfIndex": e.defaultIfIndex, "routeErr": fmt.Sprintf("%v", routeErr), "routeOut": strings.TrimSpace(routeOut)})
	// #endregion
}

func (e *Engine) runHostRouteDiagnostic(testNetwork, testTarget, trigger, failingTarget, gateway string) {
	targetHost := hostOnly(testTarget)
	addOut, addErr := addHostRoute(targetHost, gateway, e.defaultIfIndex)
	hypothesisID := hostRouteHypothesisID(testNetwork, testTarget, failingTarget)
	// #region agent log
	debugLogCurrent(debugRunInvestigation, hypothesisID, "windows.go:runHostRouteDiagnostic", "Host route add result", map[string]interface{}{"trigger": trigger, "failingTarget": failingTarget, "testNetwork": testNetwork, "testTarget": testTarget, "targetHost": targetHost, "gateway": gateway, "defaultIfIndex": e.defaultIfIndex, "routeAddErr": fmt.Sprintf("%v", addErr), "routeAddOut": strings.TrimSpace(addOut)})
	// #endregion
	if addErr != nil {
		return
	}
	defer func() {
		delOut, delErr := deleteHostRoute(targetHost, gateway, e.defaultIfIndex)
		// #region agent log
		debugLogCurrent(debugRunInvestigation, hypothesisID, "windows.go:runHostRouteDiagnostic", "Host route delete result", map[string]interface{}{"trigger": trigger, "failingTarget": failingTarget, "testNetwork": testNetwork, "testTarget": testTarget, "targetHost": targetHost, "gateway": gateway, "defaultIfIndex": e.defaultIfIndex, "routeDeleteErr": fmt.Sprintf("%v", delErr), "routeDeleteOut": strings.TrimSpace(delOut)})
		// #endregion
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, testNetwork, testTarget)
	localAddr := ""
	remoteAddr := ""
	writeErr := ""
	if conn != nil {
		localAddr = debugAddrString(conn.LocalAddr())
		remoteAddr = debugAddrString(conn.RemoteAddr())
		if strings.HasPrefix(testNetwork, "udp") {
			_, udpWriteErr := conn.Write([]byte{0})
			writeErr = fmt.Sprintf("%v", udpWriteErr)
		}
		conn.Close()
	}
	// #region agent log
	debugLogCurrent(debugRunInvestigation, hypothesisID, "windows.go:runHostRouteDiagnostic", "Host route dial result", map[string]interface{}{"trigger": trigger, "failingTarget": failingTarget, "testNetwork": testNetwork, "testTarget": testTarget, "targetHost": targetHost, "gateway": gateway, "defaultIfIndex": e.defaultIfIndex, "localAddr": localAddr, "remoteAddr": remoteAddr, "err": fmt.Sprintf("%v", err), "writeErr": writeErr})
	// #endregion
}

func (e *Engine) runDirectSocketDiagnostics(trigger, failingTarget string) {
	const (
		diagTimeout   = 1500 * time.Millisecond
		tcpDiagTarget = "1.1.1.1:443"
	)
	udpDiagTarget := e.dnsServer.Upstream()
	tests := []struct {
		network      string
		target       string
		hypothesisID string
	}{
		{network: "udp4", target: udpDiagTarget, hypothesisID: "H7"},
		{network: "tcp4", target: tcpDiagTarget, hypothesisID: "H7"},
	}

	// #region agent log
	debugLogCurrent(debugRunInvestigation, "H7", "windows.go:runDirectSocketDiagnostics", "Direct socket diagnostics started", map[string]interface{}{"trigger": trigger, "failingTarget": failingTarget, "udpDiagTarget": udpDiagTarget, "tcpDiagTarget": tcpDiagTarget, "localIP": debugIPString(e.localIP), "defaultIfIndex": e.defaultIfIndex})
	// #endregion

	gateway, gatewayOut, gatewayErr := getDefaultGateway(e.defaultIfIndex)
	// #region agent log
	debugLogCurrent(debugRunInvestigation, "H10", "windows.go:runDirectSocketDiagnostics", "Physical gateway discovery", map[string]interface{}{"trigger": trigger, "failingTarget": failingTarget, "defaultIfIndex": e.defaultIfIndex, "gateway": gateway, "gatewayErr": fmt.Sprintf("%v", gatewayErr), "gatewayOut": strings.TrimSpace(gatewayOut)})
	// #endregion

	for _, test := range tests {
		ctx, cancel := context.WithTimeout(context.Background(), diagTimeout)
		d := net.Dialer{}
		conn, err := d.DialContext(ctx, test.network, test.target)
		cancel()
		localAddr := ""
		remoteAddr := ""
		if conn != nil {
			localAddr = debugAddrString(conn.LocalAddr())
			remoteAddr = debugAddrString(conn.RemoteAddr())
			conn.Close()
		}
		// #region agent log
		debugLogCurrent(debugRunInvestigation, test.hypothesisID, "windows.go:runDirectSocketDiagnostics", "Direct socket diagnostic", map[string]interface{}{"trigger": trigger, "failingTarget": failingTarget, "testNetwork": test.network, "testTarget": test.target, "strategy": "plain", "localAddr": localAddr, "remoteAddr": remoteAddr, "err": fmt.Sprintf("%v", err)})
		// #endregion
		if gatewayErr == nil && gateway != "" {
			e.runHostRouteDiagnostic(test.network, test.target, trigger, failingTarget, gateway)
		}
	}
	if trigger == "tcp-direct" && gatewayErr == nil && gateway != "" && net.ParseIP(hostOnly(failingTarget)) != nil {
		e.runHostRouteDiagnostic("tcp4", failingTarget, trigger, failingTarget, gateway)
	}
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
		run("netsh", "interface", "ip", "set", "dnsservers",                  //nolint
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
