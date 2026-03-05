package proxy

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

// UDPSession 代表一个 SOCKS5 UDP ASSOCIATE 会话
// tcpConn 是控制连接，必须保持打开直到 UDP 会话结束
type UDPSession struct {
	tcpConn   net.Conn
	UDPConn   *net.UDPConn
	RelayAddr *net.UDPAddr
}

func (s *UDPSession) Close() {
	s.UDPConn.Close()
	s.tcpConn.Close()
}

// UDPAssociate 建立 SOCKS5 UDP 中继会话
func (c *Socks5Client) UDPAssociate() (*UDPSession, error) {
	// 1. 建立 TCP 控制连接
	tcpConn, err := net.DialTimeout("tcp", c.ProxyAddr, c.Timeout)
	if err != nil {
		return nil, fmt.Errorf("UDP ASSOCIATE 控制连接失败: %w", err)
	}
	tcpConn.SetDeadline(time.Now().Add(c.Timeout))

	// 2. 认证协商（无认证）
	if _, err := tcpConn.Write([]byte{socks5Version, 1, noAuth}); err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("UDP 握手失败: %w", err)
	}
	resp := make([]byte, 2)
	if _, err := io.ReadFull(tcpConn, resp); err != nil || resp[1] != noAuth {
		tcpConn.Close()
		return nil, fmt.Errorf("UDP 认证失败")
	}

	// 3. 发送 UDP ASSOCIATE 请求（0.0.0.0:0 表示接受任意来源）
	req := []byte{socks5Version, 0x03, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0}
	if _, err := tcpConn.Write(req); err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("UDP ASSOCIATE 请求失败: %w", err)
	}

	// 4. 解析代理返回的 UDP 中继地址
	header := make([]byte, 4)
	if _, err := io.ReadFull(tcpConn, header); err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("UDP ASSOCIATE 响应失败: %w", err)
	}
	if header[1] != socks5Success {
		tcpConn.Close()
		return nil, fmt.Errorf("UDP ASSOCIATE 被拒绝: 错误码 0x%02x", header[1])
	}

	relayHost, relayPort, err := readSocksAddr(tcpConn, header[3])
	if err != nil {
		tcpConn.Close()
		return nil, err
	}
	tcpConn.SetDeadline(time.Time{})

	// 5. 创建本地 UDP socket 发送数据到中继地址
	udpConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("创建本地 UDP socket 失败: %w", err)
	}

	relayAddr := &net.UDPAddr{
		IP:   net.ParseIP(relayHost),
		Port: int(relayPort),
	}

	return &UDPSession{
		tcpConn:   tcpConn,
		UDPConn:   udpConn,
		RelayAddr: relayAddr,
	}, nil
}

// SendUDP 通过 SOCKS5 UDP 中继发送一个 datagram
// targetHost/Port 是最终目标（域名或 IP）
func (s *UDPSession) SendUDP(data []byte, targetHost string, targetPort uint16) error {
	pkt, err := buildUDPPacket(data, targetHost, targetPort)
	if err != nil {
		return err
	}
	_, err = s.UDPConn.WriteToUDP(pkt, s.RelayAddr)
	return err
}

// RecvUDP 从 SOCKS5 UDP 中继接收一个 datagram，返回 payload 和来源地址
func (s *UDPSession) RecvUDP(buf []byte) (payload []byte, srcHost string, srcPort uint16, err error) {
	n, _, err := s.UDPConn.ReadFromUDP(buf)
	if err != nil {
		return nil, "", 0, err
	}
	return parseUDPPacket(buf[:n])
}

// buildUDPPacket 构造 SOCKS5 UDP 包头
// +----+------+------+----------+----------+----------+
// |RSV | FRAG | ATYP | DST.ADDR | DST.PORT |   DATA   |
// +----+------+------+----------+----------+----------+
// |  2 |   1  |   1  | Variable |    2     | Variable |
func buildUDPPacket(data []byte, host string, port uint16) ([]byte, error) {
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, port)

	var header []byte
	ip := net.ParseIP(host)
	if ip4 := ip.To4(); ip4 != nil {
		header = append([]byte{0, 0, 0, atypIPv4}, ip4...)
	} else if ip != nil {
		header = append([]byte{0, 0, 0, atypIPv6}, ip.To16()...)
	} else {
		if len(host) > 255 {
			return nil, fmt.Errorf("域名过长")
		}
		header = append([]byte{0, 0, 0, atypDomain, byte(len(host))}, []byte(host)...)
	}
	header = append(header, portBytes...)
	return append(header, data...), nil
}

// parseUDPPacket 解析 SOCKS5 UDP 包头，返回 payload 和来源地址
func parseUDPPacket(pkt []byte) (payload []byte, host string, port uint16, err error) {
	if len(pkt) < 4 {
		return nil, "", 0, fmt.Errorf("UDP 包过短")
	}
	// pkt[0..1] = RSV, pkt[2] = FRAG（忽略分片）
	atyp := pkt[3]
	rest := pkt[4:]

	switch atyp {
	case atypIPv4:
		if len(rest) < 4+2 {
			return nil, "", 0, fmt.Errorf("IPv4 UDP 包截断")
		}
		host = net.IP(rest[:4]).String()
		port = binary.BigEndian.Uint16(rest[4:6])
		payload = rest[6:]
	case atypIPv6:
		if len(rest) < 16+2 {
			return nil, "", 0, fmt.Errorf("IPv6 UDP 包截断")
		}
		host = net.IP(rest[:16]).String()
		port = binary.BigEndian.Uint16(rest[16:18])
		payload = rest[18:]
	case atypDomain:
		if len(rest) < 1 {
			return nil, "", 0, fmt.Errorf("域名 UDP 包截断")
		}
		dLen := int(rest[0])
		if len(rest) < 1+dLen+2 {
			return nil, "", 0, fmt.Errorf("域名 UDP 包截断")
		}
		host = string(rest[1 : 1+dLen])
		port = binary.BigEndian.Uint16(rest[1+dLen : 1+dLen+2])
		payload = rest[1+dLen+2:]
	default:
		return nil, "", 0, fmt.Errorf("未知 UDP ATYP: 0x%02x", atyp)
	}
	return payload, host, port, nil
}

func readSocksAddr(r io.Reader, atyp byte) (host string, port uint16, err error) {
	portBuf := make([]byte, 2)
	switch atyp {
	case atypIPv4:
		buf := make([]byte, 4)
		if _, err = io.ReadFull(r, buf); err != nil {
			return
		}
		host = net.IP(buf).String()
	case atypIPv6:
		buf := make([]byte, 16)
		if _, err = io.ReadFull(r, buf); err != nil {
			return
		}
		host = net.IP(buf).String()
	case atypDomain:
		lb := make([]byte, 1)
		if _, err = io.ReadFull(r, lb); err != nil {
			return
		}
		buf := make([]byte, lb[0])
		if _, err = io.ReadFull(r, buf); err != nil {
			return
		}
		host = string(buf)
	default:
		err = fmt.Errorf("未知 ATYP: 0x%02x", atyp)
		return
	}
	if _, err = io.ReadFull(r, portBuf); err != nil {
		return
	}
	port = binary.BigEndian.Uint16(portBuf)
	return
}
