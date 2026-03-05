# Aether

> TUN 模式透明代理 —— 无需 DLL 注入，对游戏等程序完全透明

## 特性

- **TUN 虚拟网卡**：基于 wintun（WireGuard 同款驱动），流量在进程外拦截
- **用户态 TCP/IP 栈**：gVisor netstack，sing-box/Clash Meta 同款
- **FakeIP DNS**：拦截 DNS 查询，还原域名后走 SOCKS5 域名代理
- **路由规则**：按进程名 / IP / 域名灵活分流（proxy / direct / block）
- **SOCKS5 代理**：连接本地代理客户端（V2RayN / Clash 等）

## 快速开始

### 环境要求

- Windows 10/11 x64
- 管理员权限（wintun 驱动需要）
- 本地 SOCKS5 代理（如 V2RayN 10808 端口）

### 运行

```bash
# 以管理员身份运行
aether.exe -config config.json
```

### 配置示例

```json
{
  "proxy": { "host": "127.0.0.1", "port": 10808, "type": "socks5" },
  "tun": {
    "adapter_name": "Aether-TUN",
    "address": "198.18.0.1/15",
    "dns_listen": "198.18.0.2:53",
    "mtu": 9000,
    "auto_route": true
  },
  "dns": { "mode": "fakeip", "fakeip_cidr": "198.18.0.0/15", "upstream": "8.8.8.8:53" },
  "routing": {
    "default_action": "proxy",
    "use_default_private": true,
    "rules": [
      { "type": "process", "match": "game.exe", "action": "proxy" }
    ]
  }
}
```

## 开发

```bash
go build -o aether.exe .
```

## 与 antigravity-proxy 对比

| | antigravity-proxy | Aether |
|--|--|--|
| 原理 | DLL 注入 + API Hook | TUN 虚拟网卡 |
| 反作弊兼容 | 差 | 好 |
| 需要管理员权限 | 否 | 是 |
| GUI | 无（计划中） | 计划中 |

## 路线图

- [x] Phase 1: TUN 核心（命令行验证）
- [ ] Phase 2: 规则引擎 + DNS 完善
- [ ] Phase 3: Wails GUI
- [ ] Phase 4: 系统托盘 + 打包
