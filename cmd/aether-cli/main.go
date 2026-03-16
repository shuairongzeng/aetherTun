package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/shuairongzeng/aether/internal/runtime"
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

	fmt.Printf("Aether CLI v%s 启动中...\n", version)

	factory := runtime.NewLiveFactory(*configPath)
	manager := runtime.NewManager(factory)
	if err := manager.Start(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "启动运行时失败: %v\n", err)
		os.Exit(1)
	}
	defer manager.Stop(context.Background())

	cfg, err := factory.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("代理: %s://%s:%d\n", cfg.Proxy.Type, cfg.Proxy.Host, cfg.Proxy.Port)
	fmt.Printf("TUN: %s (%s)\n", cfg.Tun.AdapterName, cfg.Tun.Address)
	fmt.Println("按 Ctrl+C 退出")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("正在关闭...")
}
