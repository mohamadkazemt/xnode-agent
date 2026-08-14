package xray

import "testing"

func TestParseOnlineSessions(t *testing.T) {
	in := []byte(`{"users":[{"email":"u:25|i:101","ips":[{"ip":"203.0.113.2","lastSeen":"123"},{"ip":"203.0.113.1","lastSeen":"124"}]}]}`)
	got, err := ParseOnlineSessions(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].UserID != "25" || got[0].InboundID != "101" || len(got[0].IPs) != 2 || got[0].LastSeen != 124 {
		t.Fatalf("unexpected %#v", got)
	}
	if got[0].Source != "xray-online" {
		t.Fatalf("source=%q", got[0].Source)
	}
}
