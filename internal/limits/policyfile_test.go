package limits

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"xnode-agent/internal/model"
)

func TestWriteCorePolicy(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "limits.json")
	st := model.DesiredState{Mode: "draining", Inbounds: []model.ManagedInbound{{ID: "101", Users: []model.ManagedUser{{ID: "25", Enabled: true, Limits: model.UserLimits{UploadBPS: 100, DownloadBPS: 200, ConnectionLimit: 3, IPLimit: 2}}}}}}
	if err := WriteCorePolicy(p, st, time.Unix(10, 0)); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var d CorePolicyFile
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	if d.Mode != "draining" {
		t.Fatalf("mode=%q", d.Mode)
	}
	got := d.Users["u:25|i:101"]
	if got.UploadBPS != 100 || got.DownloadBPS != 200 || got.ConnectionLimit != 3 || got.IPLimit != 2 {
		t.Fatalf("got %#v", got)
	}
	info, _ := os.Stat(p)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o", info.Mode().Perm())
	}
}

func TestCorePolicyKeepsRemovedUserTombstone(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "limits.json")
	now := time.Unix(1000, 0)
	initial := model.DesiredState{Inbounds: []model.ManagedInbound{{ID: "101", Users: []model.ManagedUser{{ID: "25", Enabled: true, SessionGeneration: 7}}}}}
	if err := WriteCorePolicy(p, initial, now); err != nil {
		t.Fatal(err)
	}
	if err := WriteCorePolicy(p, model.DesiredState{}, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var d CorePolicyFile
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	got, ok := d.Users["u:25|i:101"]
	if !ok || !got.Blocked || got.TombstoneUntil <= now.Unix() || got.SessionGeneration != 7 {
		t.Fatalf("unexpected tombstone: %#v", got)
	}
	strict := StrictPolicyEmailsFromFile(p, now.Add(time.Second))
	if len(strict) != 1 || strict[0] != "u:25|i:101" {
		t.Fatalf("strict=%v", strict)
	}
}

func TestCorePolicyDropsExpiredTombstone(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "limits.json")
	now := time.Unix(2000, 0)
	doc := CorePolicyFile{Version: 1, Users: map[string]CorePolicy{"u:old|i:1": {Blocked: true, TombstoneUntil: now.Add(-time.Second).Unix()}}}
	b, _ := json.Marshal(doc)
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteCorePolicy(p, model.DesiredState{}, now); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(p)
	var got CorePolicyFile
	_ = json.Unmarshal(b, &got)
	if _, exists := got.Users["u:old|i:1"]; exists {
		t.Fatalf("expired tombstone retained: %#v", got.Users)
	}
}

func TestWriteCorePolicyDisablesIPLimitWhenInboundModeOff(t *testing.T) {
	p := filepath.Join(t.TempDir(), "limits.json")
	st := model.DesiredState{Inbounds: []model.ManagedInbound{{ID: "101", IPLimitMode: "off", Users: []model.ManagedUser{{ID: "25", Enabled: true, Limits: model.UserLimits{IPLimit: 2}}}}}}
	if err := WriteCorePolicy(p, st, time.Now()); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	var d CorePolicyFile
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	if got := d.Users["u:25|i:101"].IPLimit; got != 0 {
		t.Fatalf("ip limit=%d", got)
	}
}
