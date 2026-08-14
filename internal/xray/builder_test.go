package xray

import (
	"encoding/json"
	"testing"
	"xnode-agent/internal/model"
)

func TestBuildVLESSAccounting(t *testing.T) {
	s := model.DesiredState{Enabled: true, Inbounds: []model.ManagedInbound{{ID: "101", Tag: "vless-443", Protocol: "vless", Port: 443, Users: []model.ManagedUser{{ID: "25", Enabled: true, Credential: map[string]any{"id": "uuid"}}}}}}
	b, err := BuildConfig(s, "127.0.0.1:10085")
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	ins := cfg["inbounds"].([]any)
	settings := ins[0].(map[string]any)["settings"].(map[string]any)
	clients := settings["clients"].([]any)
	if clients[0].(map[string]any)["email"] != "u:25|i:101" {
		t.Fatalf("unexpected email %#v", clients[0])
	}
}

func TestParseAccountingName(t *testing.T) {
	u, i, d, ok := ParseAccountingName("user>>>u:25|i:101>>>traffic>>>uplink")
	if !ok || u != "25" || i != "101" || d != "uplink" {
		t.Fatalf("bad parse %v %q %q %q", ok, u, i, d)
	}
}
