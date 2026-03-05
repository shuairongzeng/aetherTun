# Aether 路由配置脚本
# 以管理员身份运行，将默认流量路由到 Aether-TUN 适配器
# 用法: .\setup-route.ps1 [-Remove]

param([switch]$Remove)

$adapterName = "Aether-TUN"
$tunGateway  = "198.18.0.1"

# 找到 TUN 适配器的接口索引
$iface = Get-NetAdapter -Name $adapterName -ErrorAction SilentlyContinue
if (-not $iface) {
    Write-Error "未找到适配器 '$adapterName'，请先启动 aether.exe"
    exit 1
}
$ifIndex = $iface.InterfaceIndex

if ($Remove) {
    Write-Host "移除 Aether 路由..."
    # 删除默认路由（只删除指向 TUN 的那条）
    Remove-NetRoute -InterfaceIndex $ifIndex -DestinationPrefix "0.0.0.0/0" -Confirm:$false -ErrorAction SilentlyContinue
    Remove-NetRoute -InterfaceIndex $ifIndex -DestinationPrefix "::/0"      -Confirm:$false -ErrorAction SilentlyContinue
    # 删除 DNS 设置
    Set-DnsClientServerAddress -InterfaceIndex $ifIndex -ResetServerAddresses
    Write-Host "路由已移除"
    exit 0
}

Write-Host "配置 Aether 路由 (接口索引: $ifIndex)..."

# 设置 TUN 接口 IP（wintun 只创建适配器，需手动配置 IP）
$existingIP = Get-NetIPAddress -InterfaceIndex $ifIndex -AddressFamily IPv4 -ErrorAction SilentlyContinue
if (-not $existingIP) {
    New-NetIPAddress -InterfaceIndex $ifIndex -IPAddress $tunGateway -PrefixLength 15 | Out-Null
    Write-Host "  已设置 TUN IP: $tunGateway/15"
}

# 设置 DNS 指向 Aether FakeIP DNS（198.18.0.2）
Set-DnsClientServerAddress -InterfaceIndex $ifIndex -ServerAddresses "198.18.0.2"
Write-Host "  已设置 DNS: 198.18.0.2 (FakeIP)"

# 添加默认路由，将全部流量导入 TUN
# metric 1 确保优先级高于物理网卡
$existingRoute = Get-NetRoute -InterfaceIndex $ifIndex -DestinationPrefix "0.0.0.0/0" -ErrorAction SilentlyContinue
if (-not $existingRoute) {
    New-NetRoute -InterfaceIndex $ifIndex -DestinationPrefix "0.0.0.0/0" -NextHop $tunGateway -RouteMetric 1 | Out-Null
    Write-Host "  已添加默认路由 → $tunGateway (metric 1)"
}

Write-Host ""
Write-Host "Aether 路由配置完成！"
Write-Host "关闭时运行: .\setup-route.ps1 -Remove"
