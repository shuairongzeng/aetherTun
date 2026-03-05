package proxy

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	socks5Version    = 0x05
	noAuth           = 0x00
	cmdConnect       = 0x01
	atypIPv4         = 0x01
	atypDomain       = 0x03
	atypIPv6         = 0x04
	socks5Success    = 0x00
)

type Socks5Client struct {
	ProxyAddr string // host:port
	Timeout   time.Duration
}

func NewSocks5Client(host string, port int) *Socks5Client {
	return &Socks5Client{
		ProxyAddr: fmt.Sprintf("%s:%d", host, port),
		Timeout:   10 * time.Second,
	}
}

// Connect 通过 SOCKS5 代理建立到目标的 TCP 连接
func (c *Socks5Client) Connect(targetHost string, targetPort uint16) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", c.ProxyAddr, c.Timeout)
	if err != nil {
		return nil, fmt.Errorf("连接代理失败: %w", err)
	}

	if err := conn.SetDeadline(time.Now().Add(c.Timeout)); err != nil {
		conn.Close()
		return nil, err
	}

	// 1. 握手：协商认证方式（无认证）
	if _, err := conn.Write([]byte{socks5Version, 1, noAuth}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("握手写入失败: %w", err)
	}
	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		conn.Close()
		return nil, fmt.Errorf("握手响应失败: %w", err)
	}
	if resp[0] != socks5Version || resp[1] != noAuth {
		conn.Close()
		return nil, fmt.Errorf("代理不支持无认证模式")
	}

	// 2. 发送 CONNECT 请求
	req, err := buildConnectRequest(targetHost, targetPort)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := conn.Write(req); err != nil {
		conn.Close()
		return nil, fmt.Errorf("CONNECT 请求失败: %w", err)
	}

	// 3. 读取响应
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		conn.Close()
		return nil, fmt.Errorf("读取 CONNECT 响应失败: %w", err)
	}
	if header[1] != socks5Success {
		conn.Close()
		return nil, fmt.Errorf("SOCKS5 CONNECT 被拒绝: 错误码 0x%02x", header[1])
	}

	// 跳过绑定地址
	if err := skipBoundAddr(conn, header[3]); err != nil {
		conn.Close()
		return nil, err
	}

	// 清除 deadline，交给调用方控制
	conn.SetDeadline(time.Time{})
	return conn, nil
}

func buildConnectRequest(host string, port uint16) ([]byte, error) {
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, port)

	ip := net.ParseIP(host)
	if ip4 := ip.To4(); ip4 != nil {
		req := []byte{socks5Version, cmdConnect, 0x00, atypIPv4}
		req = append(req, ip4...)
		req = append(req, portBytes...)
		return req, nil
	}
	if ip6 := ip.To16(); ip != nil && ip6 != nil {
		req := []byte{socks5Version, cmdConnect, 0x00, atypIPv6}
		req = append(req, ip6...)
		req = append(req, portBytes...)
		return req, nil
	}
	// 域名
	if len(host) > 255 {
		return nil, fmt.Errorf("域名过长")
	}
	req := []byte{socks5Version, cmdConnect, 0x00, atypDomain, byte(len(host))}
	req = append(req, []byte(host)...)
	req = append(req, portBytes...)
	return req, nil
}

func skipBoundAddr(conn net.Conn, atyp byte) error {
	switch atyp {
	case atypIPv4:
		buf := make([]byte, 4+2)
		_, err := io.ReadFull(conn, buf)
		return err
	case atypIPv6:
		buf := make([]byte, 16+2)
		_, err := io.ReadFull(conn, buf)
		return err
	case atypDomain:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return err
		}
		buf := make([]byte, int(lenBuf[0])+2)
		_, err := io.ReadFull(conn, buf)
		return err
	}
	return fmt.Errorf("未知地址类型: 0x%02x", atyp)
}
