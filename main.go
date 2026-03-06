package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/shuairongzeng/aether/internal/config"
	"github.com/shuairongzeng/aether/internal/dns"
	"github.com/shuairongzeng/aether/internal/routing"
	"github.com/shuairongzeng/aether/internal/tun"
)

var version = "0.1.0-dev"

func main() {
	configPath := flag.String("config", "config.json", "配置文件路径")
	showVersion := flag.Bool("version", false, "显示版本号")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Aether v%s\n", version)
		os.Exit(0)
	}

	fmt.Printf("Aether v%s 启动中...\n", version)

	// 释放内嵌的 wintun.dll（版本 0.14.1）到 exe 同目录
	if err := extractWintunDLL(); err != nil {
		fmt.Fprintf(os.Stderr, "释放 wintun.dll 失败: %v\n", err)
		os.Exit(1)
	}

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化路由引擎
	router := routing.New(&cfg.Routing)

	// 先启动 TUN 引擎（自动配置适配器 IP，DNS 才能 bind）
	dnsServer, err := dns.NewServer(
		cfg.Tun.DNSListen,
		cfg.DNS.Upstream,
		cfg.DNS.FakeIPCIDR,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化 DNS 失败: %v\n", err)
		os.Exit(1)
	}

	engine := tun.New(cfg, dnsServer, router)
	if err := engine.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "启动 TUN 失败: %v\n", err)
		os.Exit(1)
	}
	defer engine.Stop()

	// TUN 适配器 IP 已配置，现在可以启动 DNS 服务器
	if err := dnsServer.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "启动 DNS 失败: %v\n", err)
		os.Exit(1)
	}
	defer dnsServer.Stop()

	fmt.Printf("代理: %s://%s:%d\n", cfg.Proxy.Type, cfg.Proxy.Host, cfg.Proxy.Port)
	fmt.Printf("TUN: %s (%s)\n", cfg.Tun.AdapterName, cfg.Tun.Address)
	fmt.Println("按 Ctrl+C 退出")

	// 等待退出信号
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("正在关闭...")
}
