package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"

	"xnode-agent/internal/model"
)

// QueryOnlineSessions uses Xray's native online-map API when available. This
// reports IPs that have an active dispatcher reference, which is more precise
// than access-log rolling windows. Older Xray builds return an error and the
// agent transparently falls back to access-log tracking.
func (m *Manager) QueryOnlineSessions(ctx context.Context) ([]model.SessionRecord, error) {
	args := []string{"api", "statsonlineiplist", "--server=" + m.API, "-all"}
	out, err := exec.CommandContext(ctx, m.Binary, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("statsonlineiplist: %w: %s", err, string(out))
	}
	return ParseOnlineSessions(out)
}

func ParseOnlineSessions(b []byte) ([]model.SessionRecord, error) {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	users, _ := raw["users"].([]any)
	out := make([]model.SessionRecord, 0, len(users))
	for _, item := range users {
		u, _ := item.(map[string]any)
		email, _ := u["email"].(string)
		userID, inboundID, ok := ParseAccountingEmail(email)
		if !ok {
			continue
		}
		rec := model.SessionRecord{UserID: userID, InboundID: inboundID, Source: "xray-online"}
		if ips, ok := u["ips"].([]any); ok {
			for _, iv := range ips {
				im, _ := iv.(map[string]any)
				ip, _ := im["ip"].(string)
				if ip == "" {
					continue
				}
				rec.IPs = append(rec.IPs, ip)
				if ts := anyInt64(im["lastSeen"]); ts > rec.LastSeen {
					rec.LastSeen = ts
				}
				if ts := anyInt64(im["last_seen"]); ts > rec.LastSeen {
					rec.LastSeen = ts
				}
			}
		}
		// Some protobuf JSON encoders may expose the map-style response used
		// by the single-user RPC. Accept it as a compatibility fallback.
		if len(rec.IPs) == 0 {
			if ips, ok := u["ips"].(map[string]any); ok {
				for ip, tv := range ips {
					rec.IPs = append(rec.IPs, ip)
					if ts := anyInt64(tv); ts > rec.LastSeen {
						rec.LastSeen = ts
					}
				}
			}
		}
		sort.Strings(rec.IPs)
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].InboundID == out[j].InboundID {
			return out[i].UserID < out[j].UserID
		}
		return out[i].InboundID < out[j].InboundID
	})
	return out, nil
}

func anyInt64(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	case json.Number:
		n, _ := x.Int64()
		return n
	default:
		return 0
	}
}
