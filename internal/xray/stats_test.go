package xray

import "testing"

func TestParseStats(t *testing.T) {
	b := []byte(`{"stat":[{"name":"user>>>u:25|i:101>>>traffic>>>downlink","value":"2040"}]}`)
	r, err := ParseStats(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(r) != 1 || r[0].UserID != "25" || r[0].InboundID != "101" || r[0].Value != 2040 {
		t.Fatalf("bad %#v", r)
	}
}

func TestParseStatsSkipsNullWithoutDroppingValidCounters(t *testing.T) {
	b := []byte(`{"stat":[{"name":"outbound>>>direct>>>traffic>>>uplink","value":null},{"name":"user>>>u:25|i:101>>>traffic>>>uplink","value":"512"}]}`)
	r, err := ParseStats(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(r) != 1 || r[0].Value != 512 || r[0].Direction != "uplink" {
		t.Fatalf("bad %#v", r)
	}
}
