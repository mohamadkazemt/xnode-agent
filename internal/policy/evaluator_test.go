package policy

import (
	"testing"
	"time"

	"xnode-agent/internal/model"
)

func stateWithUser(l model.UserLimits) model.DesiredState {
	return model.DesiredState{Enabled: true, Inbounds: []model.ManagedInbound{{ID: "i1", Users: []model.ManagedUser{{ID: "u1", Enabled: true, Limits: l}}}}}
}

func TestEvaluateExpiration(t *testing.T) {
	e := Evaluate(stateWithUser(model.UserLimits{ExpiresAt: 99}), nil, nil, time.Unix(100, 0))
	if e.State.Inbounds[0].Users[0].Enabled || len(e.Violations) != 1 || e.Violations[0].Reason != "expired" {
		t.Fatalf("unexpected evaluation: %#v", e)
	}
}

func TestEvaluateQuotaIncludesPending(t *testing.T) {
	s := stateWithUser(model.UserLimits{TrafficBytes: 1000, TrafficUsedBytes: 800})
	pending := map[UsageKey]int64{{UserID: "u1", InboundID: "i1"}: 250}
	e := Evaluate(s, nil, pending, time.Unix(100, 0))
	if e.State.Inbounds[0].Users[0].Enabled || e.Violations[0].Reason != "traffic_quota" {
		t.Fatalf("unexpected evaluation: %#v", e)
	}
}

func TestEvaluateIPLimit(t *testing.T) {
	s := stateWithUser(model.UserLimits{IPLimit: 1})
	sessions := []model.SessionRecord{{UserID: "u1", InboundID: "i1", IPs: []string{"1.1.1.1", "2.2.2.2"}}}
	e := Evaluate(s, sessions, nil, time.Unix(100, 0))
	if e.State.Inbounds[0].Users[0].Enabled || e.Violations[0].Reason != "ip_limit" {
		t.Fatalf("unexpected evaluation: %#v", e)
	}
}

func TestDeviceLimitByAccount(t *testing.T) {
	st := model.DesiredState{Inbounds: []model.ManagedInbound{{ID: "in", Users: []model.ManagedUser{
		{ID: "d1", AccountID: "a", Enabled: true, Limits: model.UserLimits{DeviceLimit: 2}},
		{ID: "d2", AccountID: "a", Enabled: true, Limits: model.UserLimits{DeviceLimit: 2}},
		{ID: "d3", AccountID: "a", Enabled: true, Limits: model.UserLimits{DeviceLimit: 2}},
	}}}}
	got := Evaluate(st, nil, nil, time.Now())
	if !got.State.Inbounds[0].Users[0].Enabled || !got.State.Inbounds[0].Users[1].Enabled || got.State.Inbounds[0].Users[2].Enabled {
		t.Fatalf("unexpected %#v", got.State.Inbounds[0].Users)
	}
	if len(got.Violations) != 1 || got.Violations[0].Reason != "device_limit" {
		t.Fatalf("violations %#v", got.Violations)
	}
}

func TestApplyDrainRejectsNewMembership(t *testing.T) {
	old := model.DesiredState{Inbounds: []model.ManagedInbound{{ID: "a", Users: []model.ManagedUser{{ID: "u1", Enabled: true}}}}}
	next := model.DesiredState{Inbounds: []model.ManagedInbound{{ID: "a", Users: []model.ManagedUser{{ID: "u1", Enabled: true}, {ID: "u2", Enabled: true}}}, {ID: "new"}}}
	got := ApplyDrain(old, next)
	if got.Mode != "draining" || len(got.Inbounds) != 1 || len(got.Inbounds[0].Users) != 1 || got.Inbounds[0].Users[0].ID != "u1" {
		t.Fatalf("unexpected %#v", got)
	}
}
