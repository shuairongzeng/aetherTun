# 修复代理程序运行20分钟后网络不可用的Bug

## 问题描述

程序启动后可正常代理，但运行约20分钟后DNS查询全部失败，所有出站连接失败。

## 根因

1. **`directResolver` 和 `LookupIPv4` 的 DNS socket 未绑定物理网卡**，被 TUN 默认路由捕获形成回环
2. **UDP session 驱逐/过期时未关闭 `appConn`**，导致 gVisor endpoint 和 socket 泄漏

## 修复方案

详见 implementation_plan.md
