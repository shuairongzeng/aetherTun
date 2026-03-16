//go:build windows

package tray

import (
	_ "embed"
	"time"

	"github.com/getlantern/systray"
	"github.com/shuairongzeng/aether/internal/runtime"
)

//go:embed icon.ico
var trayIcon []byte

type MenuItem struct {
	Title string
}

type MenuModel struct {
	Items []MenuItem
}

type Controller struct {
	getStatus  func() runtime.RuntimeStatus
	onStart    func()
	onStop     func()
	onOpen     func()
	onOpenLogs func()
	onExit     func()

	toggleItem *systray.MenuItem
	openItem   *systray.MenuItem
	logsItem   *systray.MenuItem
	exitItem   *systray.MenuItem
}

func BuildMenuModel(status runtime.RuntimeStatus) MenuModel {
	toggleTitle := "启动代理"
	if status.Phase == runtime.PhaseRunning || status.Phase == runtime.PhaseStarting {
		toggleTitle = "停止代理"
	}

	return MenuModel{
		Items: []MenuItem{
			{Title: toggleTitle},
			{Title: "打开 Aether"},
			{Title: "查看日志"},
			{Title: "退出"},
		},
	}
}

func NewController(
	getStatus func() runtime.RuntimeStatus,
	onStart func(),
	onStop func(),
	onOpen func(),
	onOpenLogs func(),
	onExit func(),
) *Controller {
	return &Controller{
		getStatus:  getStatus,
		onStart:    onStart,
		onStop:     onStop,
		onOpen:     onOpen,
		onOpenLogs: onOpenLogs,
		onExit:     onExit,
	}
}

func (c *Controller) Run() {
	systray.Run(c.onReady, func() {})
}

func (c *Controller) onReady() {
	systray.SetIcon(trayIconData())
	systray.SetTitle("Aether")
	systray.SetTooltip("Aether")

	model := BuildMenuModel(c.getStatus())
	c.toggleItem = systray.AddMenuItem(model.Items[0].Title, "启动或停止代理")
	c.openItem = systray.AddMenuItem(model.Items[1].Title, "打开主窗口")
	c.logsItem = systray.AddMenuItem(model.Items[2].Title, "打开日志目录")
	systray.AddSeparator()
	c.exitItem = systray.AddMenuItem(model.Items[3].Title, "退出 Aether")

	go c.refreshLoop()
	go c.clickLoop()
}

func trayIconData() []byte {
	return trayIcon
}

func (c *Controller) refreshLoop() {
	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		model := BuildMenuModel(c.getStatus())
		c.toggleItem.SetTitle(model.Items[0].Title)
	}
}

func (c *Controller) clickLoop() {
	for {
		select {
		case <-c.toggleItem.ClickedCh:
			status := c.getStatus()
			if status.Phase == runtime.PhaseRunning || status.Phase == runtime.PhaseStarting {
				c.onStop()
			} else {
				c.onStart()
			}
		case <-c.openItem.ClickedCh:
			c.onOpen()
		case <-c.logsItem.ClickedCh:
			c.onOpenLogs()
		case <-c.exitItem.ClickedCh:
			c.onExit()
			systray.Quit()
			return
		}
	}
}
