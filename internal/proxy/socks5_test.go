package proxy_test

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"testing"

	"github.com/shuairongzeng/aether/internal/proxy"
)

// mockSocks5Server 模拟一个最简 SOCKS5 服务器（仅支持无认证 + CONNECT）
func mockSocks5Server(t *testing.T) (addr string, cleanup func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleMockSocks5(conn)
		}
	}()

	return ln.Addr().String(), func() { ln.Close() }
}

func handleMockSocks5(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 512)

	// 读取握手
	n, err := conn.Read(buf)
	if err != nil || n < 2 {
		return
	}
	// 响应：无认证
	conn.Write([]byte{0x05, 0x00})

	// 读取 CONNECT 请求
	n, err = conn.Read(buf)
	if err != nil || n < 7 {
		return
	}

	// 解析目标地址（简单处理，只接受 IPv4 和域名）
	var targetAddr string
	atyp := buf[3]
	switch atyp {
	case 0x01: // IPv4
		ip := net.IP(buf[4:8])
		port := int(buf[8])<<8 | int(buf[9])
		targetAddr = net.JoinHostPort(ip.String(), fmt.Sprintf("%d", port))
	case 0x03: // 域名
		dLen := int(buf[4])
		domain := string(buf[5 : 5+dLen])
		port := int(buf[5+dLen])<<8 | int(buf[6+dLen])
		targetAddr = net.JoinHostPort(domain, fmt.Sprintf("%d", port))
	default:
		// 响应错误
		conn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	// 连接真实目标
	target, err := net.Dial("tcp", targetAddr)
	if err != nil {
		conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer target.Close()

	// 响应成功
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	// 双向转发
	done := make(chan struct{}, 2)
	go func() { io.Copy(target, conn); done <- struct{}{} }()
	go func() { io.Copy(conn, target); done <- struct{}{} }()
	<-done
}

// TestSocks5ConnectIP 通过 mock SOCKS5 服务器，用 IP 地址建立连接
func TestSocks5ConnectIP(t *testing.T) {
	// 启动 mock echo 服务器
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echoLn.Close()
	go func() {
		for {
			c, err := echoLn.Accept()
			if err != nil {
				return
			}
			go io.Copy(c, c) // echo
		}
	}()

	// 启动 mock SOCKS5
	proxyAddr, cleanup := mockSocks5Server(t)
	defer cleanup()

	// 解析 proxy host:port
	host, portStr, _ := net.SplitHostPort(proxyAddr)
	port, _ := strconv.Atoi(portStr)

	client := proxy.NewSocks5Client(host, port)
	echoHost, echoPortStr, _ := net.SplitHostPort(echoLn.Addr().String())
	echoPort, _ := strconv.Atoi(echoPortStr)

	conn, err := client.Connect(echoHost, uint16(echoPort))
	if err != nil {
		t.Fatalf("SOCKS5 Connect 失败: %v", err)
	}
	defer conn.Close()

	// 发送数据，验证 echo
	msg := []byte("hello aether")
	conn.Write(msg)
	resp := make([]byte, len(msg))
	if _, err := io.ReadFull(conn, resp); err != nil {
		t.Fatalf("读取 echo 失败: %v", err)
	}
	if string(resp) != string(msg) {
		t.Errorf("echo 数据不匹配: got %q", resp)
	}
}

// TestSocks5ConnectDomain 通过域名建立 SOCKS5 连接
func TestSocks5ConnectDomain(t *testing.T) {
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echoLn.Close()
	go func() {
		for {
			c, err := echoLn.Accept()
			if err != nil {
				return
			}
			go io.Copy(c, c)
		}
	}()

	proxyAddr, cleanup := mockSocks5Server(t)
	defer cleanup()

	host, portStr, _ := net.SplitHostPort(proxyAddr)
	port, _ := strconv.Atoi(portStr)

	client := proxy.NewSocks5Client(host, port)

	_, echoPortStr, _ := net.SplitHostPort(echoLn.Addr().String())
	echoPort, _ := strconv.Atoi(echoPortStr)

	// 用 localhost 域名（mock 服务器会解析）
	conn, err := client.Connect("localhost", uint16(echoPort))
	if err != nil {
		t.Fatalf("域名 SOCKS5 Connect 失败: %v", err)
	}
	defer conn.Close()

	msg := []byte("domain connect ok")
	conn.Write(msg)
	resp := make([]byte, len(msg))
	if _, err := io.ReadFull(conn, resp); err != nil {
		t.Fatalf("读取 echo 失败: %v", err)
	}
	if string(resp) != string(msg) {
		t.Errorf("echo 数据不匹配: got %q", resp)
	}
}
