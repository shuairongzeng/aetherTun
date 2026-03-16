package paths

import (
	"testing"
)

func TestDefaultPathsLiveUnderLocalAppData(t *testing.T) {
	t.Setenv("LOCALAPPDATA", `C:\Users\Test\AppData\Local`)

	appPaths := DefaultPaths()

	if appPaths.RootDir != `C:\Users\Test\AppData\Local\Aether` {
		t.Fatalf("expected root dir %q, got %q", `C:\Users\Test\AppData\Local\Aether`, appPaths.RootDir)
	}
	if appPaths.ConfigFile != `C:\Users\Test\AppData\Local\Aether\config.json` {
		t.Fatalf("expected config file %q, got %q", `C:\Users\Test\AppData\Local\Aether\config.json`, appPaths.ConfigFile)
	}
	if appPaths.LogDir != `C:\Users\Test\AppData\Local\Aether\logs` {
		t.Fatalf("expected log dir %q, got %q", `C:\Users\Test\AppData\Local\Aether\logs`, appPaths.LogDir)
	}
	if appPaths.RuntimeDir != `C:\Users\Test\AppData\Local\Aether\run` {
		t.Fatalf("expected runtime dir %q, got %q", `C:\Users\Test\AppData\Local\Aether\run`, appPaths.RuntimeDir)
	}
}
