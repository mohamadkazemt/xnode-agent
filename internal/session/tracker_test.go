package session

import (
	"testing"
	"time"
)

func TestParseAccessLine(t *testing.T) {
	now := time.Unix(100, 0)
	line := "2026/08/14 08:00:00 from 203.0.113.9:51234 accepted tcp:example.com:443 email: u:25|i:101\n"
	ev, ok := ParseAccessLine(line, now)
	if !ok || ev.UserID != "25" || ev.InboundID != "101" || ev.IP != "203.0.113.9" {
		t.Fatalf("unexpected event: %#v ok=%v", ev, ok)
	}
	if ev.At.Year() != 2026 || ev.At.Month() != time.August || ev.At.Day() != 14 {
		t.Fatalf("timestamp was not parsed: %v", ev.At)
	}
}

func TestTrackerWindow(t *testing.T) {
	tr := NewTracker()
	now := time.Unix(1000, 0)
	tr.Add(Event{UserID: "u", InboundID: "i", IP: "1.1.1.1", At: now.Add(-10 * time.Second)})
	tr.Add(Event{UserID: "u", InboundID: "i", IP: "2.2.2.2", At: now.Add(-200 * time.Second)})
	recs := tr.Records(now, 120*time.Second)
	if len(recs) != 1 || len(recs[0].IPs) != 1 || recs[0].IPs[0] != "1.1.1.1" {
		t.Fatalf("unexpected records: %#v", recs)
	}
}
