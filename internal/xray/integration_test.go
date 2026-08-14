package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"xnode-agent/internal/model"
)

func TestXrayE2ERuntime(t *testing.T) {
	bin := os.Getenv("XRAY_BIN")
	if bin == "" {
		t.Skip("XRAY_BIN is not set")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("XRAY_BIN: %v", err)
	}

	apiPort := freeTCPPort(t)
	inboundPort := freeTCPPort(t)
	dir := t.TempDir()
	api := fmt.Sprintf("127.0.0.1:%d", apiPort)
	limitsFile := filepath.Join(dir, "limits.json")
	if err := os.WriteFile(limitsFile, []byte(`{"version":1,"users":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	u1 := model.ManagedUser{ID: "u1", Enabled: true, Credential: map[string]any{"id": "550e8400-e29b-41d4-a716-446655440001"}}
	in := model.ManagedInbound{
		ID: "in1", Tag: "e2e-vless", Listen: "127.0.0.1", Port: inboundPort, Protocol: "vless",
		Settings: map[string]any{"decryption": "none"}, Users: []model.ManagedUser{u1},
	}
	oldState := model.DesiredState{Enabled: true, Mode: "active", Inbounds: []model.ManagedInbound{in}}
	cfg, err := BuildConfig(oldState, api, filepath.Join(dir, "access.log"))
	if err != nil {
		t.Fatal(err)
	}

	m := &Manager{Binary: bin, Config: filepath.Join(dir, "xray.json"), API: api, LimitsFile: limitsFile}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := m.ForceApply(ctx, cfg); err != nil {
		t.Fatalf("start Xray: %v", err)
	}
	defer m.Stop()

	waitUntil(t, 6*time.Second, func() bool { return m.APIHealthy(ctx) }, "Xray API did not become healthy")
	if !m.CorePatchReady() {
		t.Fatal("patched core readiness marker missing")
	}
	if got := inboundUserCount(t, ctx, bin, api, in.Tag); got != 1 {
		t.Fatalf("initial user count=%d", got)
	}

	u2 := model.ManagedUser{ID: "u2", Enabled: true, Credential: map[string]any{"id": "550e8400-e29b-41d4-a716-446655440002"}}
	newState := oldState
	newState.Inbounds = append([]model.ManagedInbound(nil), oldState.Inbounds...)
	newState.Inbounds[0].Users = append([]model.ManagedUser(nil), oldState.Inbounds[0].Users...)
	newState.Inbounds[0].Users = append(newState.Inbounds[0].Users, u2)
	plan := PlanRuntime(oldState, newState)
	if plan.RequiresRestart {
		t.Fatalf("unexpected restart plan: %#v", plan)
	}
	if err := m.ApplyRuntime(ctx, plan); err != nil {
		t.Fatalf("hot add user: %v", err)
	}
	waitUntil(t, 4*time.Second, func() bool { return inboundUserCount(t, ctx, bin, api, in.Tag) == 2 }, "hot-added user was not visible")

	plan = PlanRuntime(newState, oldState)
	if err := m.ApplyRuntime(ctx, plan); err != nil {
		t.Fatalf("hot remove user: %v", err)
	}
	waitUntil(t, 4*time.Second, func() bool { return inboundUserCount(t, ctx, bin, api, in.Tag) == 1 }, "hot-removed user was still visible")

	if sessions, err := m.QueryOnlineSessions(ctx); err != nil {
		t.Fatalf("native online API: %v", err)
	} else if len(sessions) != 0 {
		t.Fatalf("unexpected online sessions: %#v", sessions)
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func inboundUserCount(t *testing.T, ctx context.Context, bin, api, tag string) int64 {
	t.Helper()
	out, err := exec.CommandContext(ctx, bin, "api", "inboundusercount", "--server="+api, "-tag="+tag).CombinedOutput()
	if err != nil {
		t.Fatalf("inboundusercount: %v: %s", err, out)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("decode inboundusercount %q: %v", out, err)
	}
	return anyInt64(doc["count"])
}

func waitUntil(t *testing.T, timeout time.Duration, fn func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal(msg)
}
