//go:build windows

package launcher

import "testing"

func TestBuildLaunchSpecIncludesCoreFlags(t *testing.T) {
	spec := buildLaunchSpec(`C:\Program Files\Aether\aether-core.exe`, LaunchOptions{
		ConfigPath:  `C:\Users\Test\AppData\Local\Aether\config.json`,
		ControlPort: 43129,
		Token:       "abc123",
	})

	if spec.Verb != "runas" {
		t.Fatalf("expected verb %q, got %q", "runas", spec.Verb)
	}
	if spec.File != `C:\Program Files\Aether\aether-core.exe` {
		t.Fatalf("expected file %q, got %q", `C:\Program Files\Aether\aether-core.exe`, spec.File)
	}
	expectedArgs := `--config "C:\Users\Test\AppData\Local\Aether\config.json" --control-port 43129 --token abc123`
	if spec.Parameters != expectedArgs {
		t.Fatalf("expected args %q, got %q", expectedArgs, spec.Parameters)
	}
	if spec.ShowCmd != 0 {
		t.Fatalf("expected hidden show command %d, got %d", 0, spec.ShowCmd)
	}
}
