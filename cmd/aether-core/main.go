package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/shuairongzeng/aether/internal/control"
	"github.com/shuairongzeng/aether/internal/logs"
	"github.com/shuairongzeng/aether/internal/runtime"
)

type coreConfig struct {
	ConfigPath  string
	ControlPort int
	Token       string
}

type managerLogReader struct {
	manager *runtime.Manager
}

func (r managerLogReader) Recent(limit int) []logs.Entry {
	return r.manager.RecentLogs(limit)
}

type coreController struct {
	manager *runtime.Manager
	cancel  context.CancelFunc
}

func (c *coreController) Status() runtime.RuntimeStatus {
	return c.manager.Status()
}

func (c *coreController) Stop(ctx context.Context) error {
	err := c.manager.Stop(ctx)
	c.cancel()
	return err
}

func parseFlags(args []string) coreConfig {
	fs := flag.NewFlagSet("aether-core", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	cfg := coreConfig{}
	fs.StringVar(&cfg.ConfigPath, "config", "", "配置文件路径")
	fs.IntVar(&cfg.ControlPort, "control-port", 43129, "控制接口端口")
	fs.StringVar(&cfg.Token, "token", "", "控制接口令牌")
	_ = fs.Parse(args)
	return cfg
}

func main() {
	cfg := parseFlags(os.Args[1:])
	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "aether-core failed: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg coreConfig) error {
	factory := runtime.NewLiveFactory(cfg.ConfigPath)
	manager := runtime.NewManager(factory)
	restoreLogOutput := configureRuntimeLogging(manager)
	defer restoreLogOutput()

	if err := manager.Start(context.Background()); err != nil {
		return err
	}
	defer manager.Stop(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	controller := &coreController{manager: manager, cancel: cancel}
	handler := control.NewServer(controller, managerLogReader{manager: manager}, cfg.Token).Handler()
	server := &http.Server{Handler: handler}

	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.ControlPort)))
	if err != nil {
		return err
	}
	defer listener.Close()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve(listener)
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
	case received := <-sig:
		_ = received
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	}

	_ = server.Shutdown(context.Background())
	return nil
}

func configureRuntimeLogging(manager *runtime.Manager) func() {
	previousWriter := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()

	log.SetFlags(0)
	log.SetPrefix("")
	log.SetOutput(io.MultiWriter(previousWriter, manager.LogWriter(logs.LevelInfo, "core")))

	return func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	}
}
