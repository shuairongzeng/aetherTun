# Aether GUI Smoke Test

## Automated Checks

已在 `2026-03-08` 执行并通过：
- [x] `cmd /c npm run test`
- [x] `cmd /c npm run build`
- [x] `go test ./...`
- [x] `go build ./cmd/aether-core`
- [x] `go build ./cmd/aether-cli`
- [x] `go build .`
- [x] `go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build -platform windows/amd64`
- [x] `powershell -ExecutionPolicy Bypass -File scripts/package-portable.ps1`
- [x] `powershell -Command "$env:PATH = \"$env:PATH;C:\Program Files (x86)\NSIS;C:\Program Files (x86)\NSIS\Bin\"; go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build -platform windows/amd64 -nsis"`

## Manual Windows Smoke

以下项目仍需在真实 Windows 环境手工勾检：
- [ ] GUI 以普通用户身份启动成功
- [ ] 点击“启动代理”后弹出 UAC
- [ ] 主窗口状态切换为“运行中”
- [ ] 托盘菜单与主窗口状态保持一致
- [ ] 关闭主窗口后仅隐藏到托盘，不直接退出
- [ ] 从托盘执行停止后，状态恢复为“未运行”
- [ ] “打开配置文件”能定位到 `%LOCALAPPDATA%\Aether\config.json`
- [ ] “查看日志”能打开 `%LOCALAPPDATA%\Aether\logs`
- [ ] GUI 配置卡片能正确加载 `proxy.host`、`proxy.port`、`proxy.type`
- [ ] 输入非法地址/端口时，“保存配置”保持禁用并显示字段级错误提示
- [ ] 保存成功时显示绿色状态条，失败时显示红色状态条
- [ ] 代理运行中保存配置时，GUI 会提示是否立即重启代理
- [ ] 首次启动且无真实配置时，自动展示首启向导
- [ ] 点击“暂时跳过”后，主界面出现首次配置提醒条
- [ ] 点击“继续配置”后，首启向导重新打开
- [ ] 首启向导保存成功后，覆盖层关闭且提醒条消失
- [ ] 一旦基础代理配置不是默认值，下次启动不再弹出首启向导
- [ ] 安装、启动、卸载流程完整可用

## Known Limitations

- GUI v1 目前只支持编辑基础代理配置：`proxy.host`、`proxy.port`、`proxy.type`
- `tun`、`dns`、`routing`、`log_level` 仍需手动编辑配置文件
- 首启向导第一版不做连接测试，只负责把用户引导到基础代理配置入口
- “开机自启”按钮当前仍是预留入口，尚未接入真实系统注册逻辑
- 首次关闭窗口时仍以日志提示“已最小化到托盘”，暂未弹出原生通知

## Notes

- 运行时数据默认位于 `%LOCALAPPDATA%\Aether\`
- 若需发布候选版本，建议先完成上面的手工烟测，再执行一次 `wails build -platform windows/amd64 -nsis`
- 当前工作树已经验证 Wails GUI 构建链路、便携包打包链路、安装器打包链路、配置页自动化测试链路和首启向导自动化测试链路都正常
- 当前开发机的 NSIS 位于 `C:\Program Files (x86)\NSIS`，如未加到 `PATH`，需先补上该目录和 `C:\Program Files (x86)\NSIS\Bin`
