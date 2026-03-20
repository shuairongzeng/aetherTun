# 开机自启功能实现计划

## 问题描述

`ToggleAutoStart()` 在 `app.go:185` 是空实现（`return nil`），前端 QuickActionsCard 中标注为"预留功能，后续接入"。

## 方案设计

使用 Windows 注册表 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` 实现当前用户级别的开机自启，无需管理员权限。

### 技术选型

- 读写 `HKCU\...\Run` 注册表键值
- 键名：`Aether`
- 键值：当前可执行文件的绝对路径
- 使用 Go 标准库 `golang.org/x/sys/windows/registry`

## 变更文件

---

### 后端

#### [NEW] internal/autostart/autostart_windows.go

- `IsEnabled() bool` — 检查注册表中是否存在 Aether 启动项
- `Enable() error` — 写入注册表
- `Disable() error` — 移除注册表项
- `Toggle() (enabled bool, err error)` — 切换状态

#### [MODIFY] app.go

- 修改 `ToggleAutoStart()` — 调用 `autostart.Toggle()`，返回新状态
- 新增 `GetAutoStartEnabled() bool` — 调用 `autostart.IsEnabled()`

---

### 前端

#### [MODIFY] types.ts

- `BackendApi` 新增 `GetAutoStartEnabled?: () => Promise<boolean>`

#### [MODIFY] components/QuickActionsCard.tsx

- 新增 `autoStartEnabled` prop
- 开机自启按钮显示当前状态（已启用/未启用）
- hint 文本根据状态变化

#### [MODIFY] hooks/useRuntimeState.ts

- 新增 `autoStartEnabled` 状态
- 在 refresh 中查询 `GetAutoStartEnabled`
- `toggleAutoStart` 后刷新状态

#### [MODIFY] pages/OverviewPage.tsx

- 传递 `autoStartEnabled` 给 QuickActionsCard

#### [MODIFY] App.tsx

- 传递 `autoStartEnabled` 给 OverviewPage

#### [MODIFY] preview/mockBackend.ts

- 新增 `GetAutoStartEnabled` mock 实现

---

## 验证计划

### 自动化测试

```bash
cd D:\GitHub\antigravityProxy\aether
go test ./internal/autostart/... -v -run .
```

```bash
cd D:\GitHub\antigravityProxy\aether\frontend
npm run test -- --run
```

### 手动验证

1. 运行 `Aether.exe`
2. 在概览页点击"开机自启"按钮
3. 按钮应显示"✓ 已启用"
4. 打开注册表编辑器 `regedit` → `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
5. 应看到键 `Aether` 的值为 exe 路径
6. 再次点击按钮取消，注册表项应被移除
