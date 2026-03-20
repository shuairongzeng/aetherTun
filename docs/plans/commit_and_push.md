# 提交并推送代码方案设计

## 目标
根据最新代码变更，准备合适的 commit message，并将代码推送到远程 GitHub 仓库。

## 变更分析
1. **`internal/dns/server.go`**:
   - 新增 `localIP` 属性并提供 `SetLocalIP(ip net.IP)` 方法。
   - `LookupIPv4` 中 UDP 和 TCP 查询使用的 `Dialer` 均绑定到此 `localIP`。
2. **`internal/tun/engine.go`**:
   - `directResolver.Dial` (UDP) 同样绑定至 `localIP`。
   - 将解析出的物理网卡 IP 传递给 DNS Server (`dnsServer.SetLocalIP(localIP)`)。
   - 修复 UDP Session 泄漏问题：在清理过期会话（`cleanupUDPSessions`）和剔除最老会话时，新增对 `sess.appConn.Close()` 的调用。

## 提交信息 (Commit Message)
**标题**: `fix(tun/dns): 绑定物理网卡 IP 防止回环并修复 UDP 会话泄漏`

**内容**:
- DNS查询(UDP/TCP)绑定 physical interface IP, 避免流量循环被 TUN 捕获。
- `directResolver.Dial` 直接绑定物理网卡 IP 连接上游。
- 增强 UDP 资源回收：清理旧会话时主动调用 `appConn.Close()` 释放资源。

## 执行步骤
1. 执行 `git add internal/dns/server.go internal/tun/engine.go`。
2. 使用上述提交信息执行 `git commit`。
3. 执行 `git push origin main`。
4. 验证推送结果。
