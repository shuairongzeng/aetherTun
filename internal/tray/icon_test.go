package tray

import "testing"

func TestTrayIconDataIsEmbedded(t *testing.T) {
	if len(trayIconData()) == 0 {
		t.Fatal("expected embedded tray icon bytes")
	}
}
