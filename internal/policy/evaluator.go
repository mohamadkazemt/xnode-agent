package policy

import (
	"strings"
	"time"

	"xnode-agent/internal/model"
)

type UsageKey struct {
	UserID    string
	InboundID string
}

type Evaluation struct {
	State      model.DesiredState
	Violations []model.PolicyViolation
}

func Evaluate(desired model.DesiredState, sessions []model.SessionRecord, pending map[UsageKey]int64, now time.Time) Evaluation {
	out := cloneState(desired)
	sessionMap := make(map[UsageKey]model.SessionRecord, len(sessions))
	for _, s := range sessions {
		sessionMap[UsageKey{s.UserID, s.InboundID}] = s
	}
	var violations []model.PolicyViolation
	for i := range out.Inbounds {
		in := &out.Inbounds[i]
		for j := range in.Users {
			u := &in.Users[j]
			if !u.Enabled {
				continue
			}
			k := UsageKey{u.ID, in.ID}
			if u.Limits.ExpiresAt > 0 && now.Unix() >= u.Limits.ExpiresAt {
				u.Enabled = false
				violations = append(violations, model.PolicyViolation{UserID: u.ID, InboundID: in.ID, Reason: "expired", Observed: now.Unix(), Limit: u.Limits.ExpiresAt})
				continue
			}
			if u.Limits.TrafficBytes > 0 {
				used := u.Limits.TrafficUsedBytes + pending[k]
				if used >= u.Limits.TrafficBytes {
					u.Enabled = false
					violations = append(violations, model.PolicyViolation{UserID: u.ID, InboundID: in.ID, Reason: "traffic_quota", Observed: used, Limit: u.Limits.TrafficBytes})
					continue
				}
			}
			if u.Limits.IPLimit > 0 && strings.ToLower(in.IPLimitMode) != "off" {
				count := len(sessionMap[k].IPs)
				if count > u.Limits.IPLimit {
					u.Enabled = false
					violations = append(violations, model.PolicyViolation{UserID: u.ID, InboundID: in.ID, Reason: "ip_limit", Observed: int64(count), Limit: int64(u.Limits.IPLimit)})
				}
			}
		}
		// Device limit is enforceable only when the panel issues one credential
		// per device and assigns the same account_id to those credentials.
		activeByAccount := map[string]int{}
		for j := range in.Users {
			u := &in.Users[j]
			if !u.Enabled || u.AccountID == "" || u.Limits.DeviceLimit <= 0 {
				continue
			}
			activeByAccount[u.AccountID]++
			if activeByAccount[u.AccountID] > u.Limits.DeviceLimit {
				u.Enabled = false
				violations = append(violations, model.PolicyViolation{UserID: u.ID, InboundID: in.ID, Reason: "device_limit", Observed: int64(activeByAccount[u.AccountID]), Limit: int64(u.Limits.DeviceLimit)})
			}
		}
	}
	return Evaluation{State: out, Violations: violations}
}

// ApplyDrain prevents new inbounds and credentials from being introduced while
// a node is draining. Existing desired members may still be updated/disabled.
func ApplyDrain(previous, desired model.DesiredState) model.DesiredState {
	out := cloneState(desired)
	out.Mode = "draining"
	prevIn := make(map[string]model.ManagedInbound, len(previous.Inbounds))
	for _, in := range previous.Inbounds {
		prevIn[in.ID] = in
	}
	kept := make([]model.ManagedInbound, 0, len(out.Inbounds))
	for _, in := range out.Inbounds {
		old, ok := prevIn[in.ID]
		if !ok {
			continue
		}
		members := map[string]struct{}{}
		for _, u := range old.Users {
			members[u.ID] = struct{}{}
		}
		users := in.Users[:0]
		for _, u := range in.Users {
			if _, ok := members[u.ID]; ok {
				users = append(users, u)
			}
		}
		in.Users = users
		kept = append(kept, in)
	}
	out.Inbounds = kept
	return out
}

func cloneState(in model.DesiredState) model.DesiredState {
	out := in
	out.Inbounds = append([]model.ManagedInbound(nil), in.Inbounds...)
	for i := range out.Inbounds {
		out.Inbounds[i] = in.Inbounds[i]
		out.Inbounds[i].Users = append([]model.ManagedUser(nil), in.Inbounds[i].Users...)
	}
	return out
}

func PendingUsage(records []model.TrafficRecord) map[UsageKey]int64 {
	out := map[UsageKey]int64{}
	for _, r := range records {
		if r.UserID == "" || r.InboundID == "" || r.Value <= 0 {
			continue
		}
		out[UsageKey{r.UserID, r.InboundID}] += r.Value
	}
	return out
}
func MergeUsage(dst map[UsageKey]int64, src map[UsageKey]int64) {
	for k, v := range src {
		dst[k] += v
	}
}
