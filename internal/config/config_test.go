package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCfg(t *testing.T, panelURL string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "agent.json")
	b := []byte(`{"node_id":"n1","panel_url":"` + panelURL + `","panel_token":"secret"}`)
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPanelURLRequiresTLS(t *testing.T) {
	if _, err := Load(writeCfg(t, "http://example.com")); err == nil {
		t.Fatal("expected non-loopback HTTP to be rejected")
	}
	if _, err := Load(writeCfg(t, "http://localhost.evil.example")); err == nil {
		t.Fatal("expected localhost prefix spoof to be rejected")
	}
	if _, err := Load(writeCfg(t, "https://panel.example.com")); err != nil {
		t.Fatalf("https rejected: %v", err)
	}
	if _, err := Load(writeCfg(t, "http://127.0.0.1:8080")); err != nil {
		t.Fatalf("loopback HTTP rejected: %v", err)
	}
}
