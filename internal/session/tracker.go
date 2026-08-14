package session

import (
	"bufio"
	"io"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"xnode-agent/internal/model"
	"xnode-agent/internal/xray"
)

type Event struct {
	UserID    string
	InboundID string
	IP        string
	At        time.Time
}

type key struct {
	userID    string
	inboundID string
}

type entry struct {
	ips  map[string]time.Time
	seen []time.Time
	last time.Time
}

type Tracker struct {
	mu       sync.Mutex
	items    map[key]*entry
	offset   int64
	lastInfo os.FileInfo
}

const maxInitialTail int64 = 8 << 20

func NewTracker() *Tracker { return &Tracker{items: make(map[key]*entry)} }

// ParseAccessLine understands Xray's AccessMessage string format. Log prefixes
// (timestamp/severity) are tolerated; only accepted records with xnode's
// deterministic accounting email are tracked.
func ParseAccessLine(line string, now time.Time) (Event, bool) {
	fromAt := strings.Index(line, "from ")
	if fromAt < 0 {
		return Event{}, false
	}
	rest := line[fromAt+len("from "):]
	acceptedAt := strings.Index(rest, " accepted ")
	if acceptedAt < 0 {
		return Event{}, false
	}
	source := strings.TrimSpace(rest[:acceptedAt])
	emailAt := strings.LastIndex(rest, " email: ")
	if emailAt < 0 {
		return Event{}, false
	}
	email := strings.TrimSpace(rest[emailAt+len(" email: "):])
	if i := strings.IndexAny(email, " \t\r\n"); i >= 0 {
		email = email[:i]
	}
	userID, inboundID, ok := xray.ParseAccountingEmail(email)
	if !ok {
		return Event{}, false
	}
	ip := sourceIP(source)
	if ip == "" {
		return Event{}, false
	}
	eventAt := parseLogTime(line[:fromAt], now)
	return Event{UserID: userID, InboundID: inboundID, IP: ip, At: eventAt}, true
}

func parseLogTime(prefix string, fallback time.Time) time.Time {
	prefix = strings.TrimSpace(prefix)
	if len(prefix) >= 19 {
		if ts, err := time.ParseInLocation("2006/01/02 15:04:05", prefix[:19], time.Local); err == nil {
			return ts
		}
	}
	return fallback
}

func sourceIP(source string) string {
	for _, prefix := range []string{"tcp:", "udp:"} {
		source = strings.TrimPrefix(source, prefix)
	}
	if host, _, err := net.SplitHostPort(source); err == nil {
		return strings.Trim(host, "[]")
	}
	if net.ParseIP(strings.Trim(source, "[]")) != nil {
		return strings.Trim(source, "[]")
	}
	return ""
}

func (t *Tracker) Add(ev Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	k := key{ev.UserID, ev.InboundID}
	e := t.items[k]
	if e == nil {
		e = &entry{ips: make(map[string]time.Time)}
		t.items[k] = e
	}
	e.ips[ev.IP] = ev.At
	e.seen = append(e.seen, ev.At)
	if ev.At.After(e.last) {
		e.last = ev.At
	}
}

// ConsumeFile consumes complete new lines since the previous call. If Xray
// truncates/rotates the file, the offset safely resets to zero.
func (t *Tracker) ConsumeFile(path string, now time.Time) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	t.mu.Lock()
	offset := t.offset
	rotated := t.lastInfo != nil && !os.SameFile(t.lastInfo, st)
	if rotated || st.Size() < offset {
		offset = 0
		t.offset = 0
	}
	initialTail := offset == 0 && st.Size() > maxInitialTail
	if initialTail {
		offset = st.Size() - maxInitialTail
	}
	t.lastInfo = st
	t.mu.Unlock()
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	r := bufio.NewReader(f)
	consumed := offset
	if initialTail {
		// A bounded tail can start in the middle of a line.
		partial, _ := r.ReadString('\n')
		consumed += int64(len(partial))
	}
	for {
		line, err := r.ReadString('\n')
		if strings.HasSuffix(line, "\n") {
			consumed += int64(len(line))
			if ev, ok := ParseAccessLine(line, now); ok {
				t.Add(ev)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	t.mu.Lock()
	t.offset = consumed
	t.mu.Unlock()
	return nil
}

func (t *Tracker) Records(now time.Time, window time.Duration) []model.SessionRecord {
	t.mu.Lock()
	defer t.mu.Unlock()
	cutoff := now.Add(-window)
	out := make([]model.SessionRecord, 0, len(t.items))
	for k, e := range t.items {
		for ip, ts := range e.ips {
			if ts.Before(cutoff) {
				delete(e.ips, ip)
			}
		}
		kept := e.seen[:0]
		for _, ts := range e.seen {
			if !ts.Before(cutoff) {
				kept = append(kept, ts)
			}
		}
		e.seen = kept
		if len(e.ips) == 0 && len(e.seen) == 0 {
			delete(t.items, k)
			continue
		}
		ips := make([]string, 0, len(e.ips))
		for ip := range e.ips {
			ips = append(ips, ip)
		}
		sort.Strings(ips)
		out = append(out, model.SessionRecord{UserID: k.userID, InboundID: k.inboundID, IPs: ips, LastSeen: e.last.Unix(), RecentConnections: len(e.seen)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].InboundID == out[j].InboundID {
			return out[i].UserID < out[j].UserID
		}
		return out[i].InboundID < out[j].InboundID
	})
	return out
}

func Counts(records []model.SessionRecord) (onlineUsers, trackedIPs int) {
	users := map[string]struct{}{}
	for _, r := range records {
		if len(r.IPs) > 0 {
			users[r.UserID] = struct{}{}
		}
		trackedIPs += len(r.IPs)
	}
	return len(users), trackedIPs
}
