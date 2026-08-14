package traffic

import (
	"testing"

	"xnode-agent/internal/model"
	"xnode-agent/internal/policy"
)

func TestSpoolRoundTrip(t *testing.T) {
	s := Spool{Dir: t.TempDir()}
	b := model.TrafficBatch{EventID: "e1", NodeID: "n", Records: []model.TrafficRecord{{UserID: "u", InboundID: "i", Value: 42}}}
	if err := s.Enqueue(b); err != nil {
		t.Fatal(err)
	}
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].EventID != "e1" {
		t.Fatalf("unexpected list: %#v", list)
	}
	usage, err := s.PendingUsage()
	if err != nil {
		t.Fatal(err)
	}
	if usage[policy.UsageKey{UserID: "u", InboundID: "i"}] != 42 {
		t.Fatalf("unexpected usage: %#v", usage)
	}
	if err := s.Delete("e1"); err != nil {
		t.Fatal(err)
	}
}
