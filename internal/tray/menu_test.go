package tray

import (
	"strings"
	"testing"

	"github.com/shuairongzeng/aether/internal/runtime"
)

func TestMenuModelReflectsRunningState(t *testing.T) {
	model := BuildMenuModel(runtime.RuntimeStatus{Phase: runtime.PhaseRunning})

	if len(model.Items) == 0 {
		t.Fatal("expected tray menu items, got none")
	}
	if !strings.Contains(model.Items[0].Title, "停止代理") {
		t.Fatalf("expected first item to contain %q, got %q", "停止代理", model.Items[0].Title)
	}
}
