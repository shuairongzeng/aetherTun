package dns

import (
	"math/rand"
	"net"
	"sync"
)

// FakeIPMap 管理域名 ↔ FakeIP 双向映射
type FakeIPMap struct {
	mu         sync.RWMutex
	domainToIP map[string]net.IP
	ipToDomain map[string]string
	pool       *ipPool
}

type ipPool struct {
	network *net.IPNet
	current uint32
	min     uint32
	max     uint32
}

func newIPPool(cidr string) (*ipPool, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	// 网段起始 IP（跳过网络地址和 .1 网关）
	base := ipToUint32(ipNet.IP)
	ones, bits := ipNet.Mask.Size()
	size := uint32(1) << uint(bits-ones)

	return &ipPool{
		network: ipNet,
		current: base + 3, // 从 .3 开始分配，.1 给 TUN 网关，.2 给 DNS
		min:     base + 3,
		max:     base + size - 1,
	}, nil
}

func (p *ipPool) next() net.IP {
	if p.current >= p.max {
		p.current = p.min // 循环分配
	}
	ip := uint32ToIP(p.current)
	p.current++
	return ip
}

func NewFakeIPMap(cidr string) (*FakeIPMap, error) {
	pool, err := newIPPool(cidr)
	if err != nil {
		return nil, err
	}
	// 随机起始偏移，避免重启后立即复用
	pool.current += uint32(rand.Intn(1000))

	return &FakeIPMap{
		domainToIP: make(map[string]net.IP),
		ipToDomain: make(map[string]string),
		pool:       pool,
	}, nil
}

// Assign 为域名分配（或复用）FakeIP
func (m *FakeIPMap) Assign(domain string) net.IP {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ip, ok := m.domainToIP[domain]; ok {
		return ip
	}
	ip := m.pool.next()
	m.domainToIP[domain] = ip
	m.ipToDomain[ip.String()] = domain
	return ip
}

// LookupDomain 通过 FakeIP 查找原始域名
func (m *FakeIPMap) LookupDomain(ip net.IP) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	domain, ok := m.ipToDomain[ip.String()]
	return domain, ok
}

// IsFakeIP 判断是否是 FakeIP 段内的地址
func (m *FakeIPMap) IsFakeIP(ip net.IP) bool {
	return m.pool.network.Contains(ip)
}

func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func uint32ToIP(n uint32) net.IP {
	return net.IP{byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}
}
