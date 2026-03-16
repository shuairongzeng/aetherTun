# Aether

> 基于 Windows TUN 的透明代理，现已提供适合新手用户的桌面 GUI / 托盘控制 MVP。

## 当前能力

- GUI 主窗口显示运行状态、最近日志和主要操作按钮
- 点击“启动代理”时由 GUI 拉起提权后的 `aether-core`
- 关闭主窗口会最小化到系统托盘，支持托盘继续控制
- 托盘菜单支持显示窗口、启动 / 停止、打开日志目录、退出
- 保留命令行入口，便于调试、回归和无界面使用

## 运行要求

- Windows 10/11 x64
- 本机可用的 SOCKS5 代理
- 首次启动核心能力时允许 UAC 提权

## 新手快速使用

1. 启动 `Aether.exe`，此时主窗口无需管理员权限。
2. 在主窗口点击“启动代理”。
3. Windows 弹出 UAC 后确认，GUI 会拉起提权后的 `aether-core`。
4. 状态卡片显示运行中后，即可保持窗口打开或直接关闭到托盘。
5. 需要停止时，可在主窗口或托盘菜单点击停止；需要完全退出时，使用托盘“退出”。

## 从源码构建

### 构建 GUI

```bash
cd frontend
cmd /c npm install
cmd /c npm run build
cd ..
go build .
```

## 便携包生成

先生成最新 GUI 二进制：

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build -platform windows/amd64
```

再执行便携打包脚本：

```bash
powershell -ExecutionPolicy Bypass -File scripts/package-portable.ps1
```

脚本会生成：
- `build/bin/Aether-portable/`
- `build/bin/Aether-portable.zip`

便携包默认包含：
- `Aether.exe`
- `aether-core.exe`
- `wintun.dll`
- `config.example.json`
- `README.txt`

注意：程序实际读取的正式配置仍是 `%LOCALAPPDATA%\Aether\config.json`；便携包内的 `config.example.json` 仅用于给用户参考或手动复制。

### 构建命令行与核心进程

```bash
go build ./cmd/aether-cli
go build ./cmd/aether-core
```

### 开发验证

```bash
cd frontend
cmd /c npm run test
cmd /c npm run build
cd ..
go test ./...
go build .
go build ./cmd/aether-core
go build ./cmd/aether-cli
```

## 运行时数据目录

- 配置文件：`%LOCALAPPDATA%\Aether\config.json`
- 日志目录：`%LOCALAPPDATA%\Aether\logs`
- 运行时目录：`%LOCALAPPDATA%\Aether\run`

GUI 会在需要时自动创建这些目录；首次打开配置文件时，也会自动生成默认配置。

## 打包安装器

先安装 Wails CLI：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

然后在仓库根目录执行：

```bash
wails build -platform windows/amd64 -nsis
```

默认产物位于 `build/bin`。当前配置面向 Windows GUI MVP，可直接用于安装器验证和手工分发。

如需生成 NSIS 安装器，请先确保系统已安装 NSIS，并且 `makensis` 在 `PATH` 中；如果本机未全局安装 `wails` CLI，也可以直接执行：

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build -platform windows/amd64 -nsis
```

如果 NSIS 安装在默认目录但尚未加入 `PATH`，可以先在 PowerShell 会话里补上：

```powershell
$env:PATH = "$env:PATH;C:\Program Files (x86)\NSIS;C:\Program Files (x86)\NSIS\Bin"
```

在当前开发机上，以上命令已于 `2026-03-08` 验证通过，并产出：

- `build/bin/Aether.exe`
- `build/bin/aether-core.exe`
- `build/bin/wintun.dll`
- `build/bin/Aether-amd64-installer.exe`

## GUI MVP 已知边界

- v1 不提供 GUI 内的完整配置编辑器
- “开机自启”入口已预留，当前版本尚未接入真实注册逻辑
- 首次关闭窗口时会写入日志提示“已最小化到托盘”，暂未弹出原生气泡提示

## 传统配置示例

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
  "dns": {
    "mode": "fakeip",
    "fakeip_cidr": "198.18.0.0/15",
    "upstream": "8.8.8.8:53"
  },
  "routing": {
    "default_action": "proxy",
    "use_default_private": true,
    "rules": [
      { "type": "process", "match": "game.exe", "action": "proxy" }
    ]
  }
}
```

## GUI 基础代理配置

当前 GUI 已内置“基础代理配置”卡片，可直接编辑以下字段：

- `proxy.host`
- `proxy.port`
- `proxy.type`

保存后的正式配置文件仍位于 `%LOCALAPPDATA%\Aether\config.json`。GUI 保存时只会更新上述三个 `proxy.*` 字段，不会覆盖 `tun`、`dns`、`routing`、`log_level` 等高级配置。

如果代理当前正在运行，保存成功后 GUI 会提示是否立即重启代理；确认后会执行一次“停止 -> 启动”，让新配置尽快生效。若你选择暂不重启，则新配置会在下次启动代理时生效。

当前 GUI 仍然不提供高级网络参数编辑；如需修改 `tun`、`dns`、`routing` 等内容，请继续通过“打开配置文件”手动编辑。

## 首启向导

当前 GUI 已内置首启向导。满足以下任一条件时，启动后会自动展示覆盖式引导层：

- `%LOCALAPPDATA%\Aether\config.json` 不存在
- `proxy.host`、`proxy.port`、`proxy.type` 仍等于默认值

首启向导第一版支持：

- 查看欢迎说明
- 填写基础代理配置
- 暂时跳过，稍后继续配置

如果你选择跳过，主界面会继续显示“尚未完成首次代理配置”的提醒条，直到基础代理配置不再是默认值。首启向导本身不做连接测试，仍然建议保存后回到主界面再点击“启动代理”验证效果。
