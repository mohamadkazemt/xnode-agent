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

func TestBuildConfigAccessLog(t *testing.T) {
	b, err := BuildConfig(model.DesiredState{}, "127.0.0.1:10085", "/var/log/xnode/access.log")
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	logCfg := cfg["log"].(map[string]any)
	if logCfg["access"] != "/var/log/xnode/access.log" {
		t.Fatalf("unexpected log config: %#v", logCfg)
	}
}

func TestParseAccountingEmail(t *testing.T) {
	u, i, ok := ParseAccountingEmail("u:25|i:101")
	if !ok || u != "25" || i != "101" {
		t.Fatalf("bad parse: %v %q %q", ok, u, i)
	}
}

func TestBuildConfigPerUserRouting(t *testing.T) {
	st := model.DesiredState{Inbounds: []model.ManagedInbound{{ID: "i1", Tag: "v", Protocol: "vless", Port: 443, Users: []model.ManagedUser{{ID: "u1", Enabled: true, OutboundTag: "premium", Credential: map[string]any{"id": "x"}}}}}, Outbounds: []json.RawMessage{json.RawMessage(`{"tag":"premium","protocol":"freedom"}`)}}
	b, err := BuildConfig(st, "127.0.0.1:10085")
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	routing := cfg["routing"].(map[string]any)
	rules := routing["rules"].([]any)
	if len(rules) != 1 {
		t.Fatalf("rules=%#v", rules)
	}
	r := rules[0].(map[string]any)
	if r["outboundTag"] != "premium" {
		t.Fatalf("rule=%#v", r)
	}
}
