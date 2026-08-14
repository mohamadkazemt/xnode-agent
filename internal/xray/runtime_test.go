package xray

import (
	"encoding/json"
	"testing"

	"xnode-agent/internal/model"
)

func testInbound(users ...model.ManagedUser) model.ManagedInbound {
	return model.ManagedInbound{ID: "in-1", Tag: "vless-443", Protocol: "vless", Port: 443, Users: users}
}

func testUser(id, uuid string, enabled bool) model.ManagedUser {
	return model.ManagedUser{ID: id, Enabled: enabled, Credential: map[string]any{"id": uuid}}
}

func TestPlanRuntimeAddUser(t *testing.T) {
	oldState := model.DesiredState{Inbounds: []model.ManagedInbound{testInbound(testUser("u1", "a", true))}}
	newState := model.DesiredState{Inbounds: []model.ManagedInbound{testInbound(testUser("u1", "a", true), testUser("u2", "b", true))}}
	plan := PlanRuntime(oldState, newState)
	if plan.RequiresRestart || len(plan.Operations) != 1 || plan.Operations[0].Kind != RuntimeAddUser {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if got := accountingEmail(plan.Operations[0].User.ID, plan.Operations[0].Inbound.ID); got != "u:u2|i:in-1" {
		t.Fatalf("unexpected accounting email %q", got)
	}
}

func TestPlanRuntimeDisableUser(t *testing.T) {
	oldState := model.DesiredState{Inbounds: []model.ManagedInbound{testInbound(testUser("u1", "a", true))}}
	newState := model.DesiredState{Inbounds: []model.ManagedInbound{testInbound(testUser("u1", "a", false))}}
	plan := PlanRuntime(oldState, newState)
	if plan.RequiresRestart || len(plan.Operations) != 1 || plan.Operations[0].Kind != RuntimeRemoveUser {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if plan.Operations[0].UserEmail != "u:u1|i:in-1" {
		t.Fatalf("unexpected email %q", plan.Operations[0].UserEmail)
	}
}

func TestPlanRuntimeCredentialChangeRemovesThenAdds(t *testing.T) {
	oldState := model.DesiredState{Inbounds: []model.ManagedInbound{testInbound(testUser("u1", "a", true))}}
	newState := model.DesiredState{Inbounds: []model.ManagedInbound{testInbound(testUser("u1", "b", true))}}
	plan := PlanRuntime(oldState, newState)
	if len(plan.Operations) != 2 || plan.Operations[0].Kind != RuntimeRemoveUser || plan.Operations[1].Kind != RuntimeAddUser {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestPlanRuntimeReplaceInboundOnStructuralChange(t *testing.T) {
	oldIn := testInbound(testUser("u1", "a", true))
	newIn := oldIn
	newIn.Port = 8443
	plan := PlanRuntime(model.DesiredState{Inbounds: []model.ManagedInbound{oldIn}}, model.DesiredState{Inbounds: []model.ManagedInbound{newIn}})
	if len(plan.Operations) != 1 || plan.Operations[0].Kind != RuntimeReplaceInbound {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestPlanRuntimeUnsupportedProtocolReplacesInbound(t *testing.T) {
	oldIn := model.ManagedInbound{ID: "wg", Tag: "wg", Protocol: "wireguard", Port: 51820, Users: []model.ManagedUser{testUser("u1", "a", true)}}
	newIn := oldIn
	newIn.Users = append(newIn.Users, testUser("u2", "b", true))
	plan := PlanRuntime(model.DesiredState{Inbounds: []model.ManagedInbound{oldIn}}, model.DesiredState{Inbounds: []model.ManagedInbound{newIn}})
	if len(plan.Operations) != 1 || plan.Operations[0].Kind != RuntimeReplaceInbound {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestPlanRuntimeGlobalChangeRequiresRestart(t *testing.T) {
	oldState := model.DesiredState{Routing: json.RawMessage(`{"rules":[]}`)}
	newState := model.DesiredState{Routing: json.RawMessage(`{"rules":[{"type":"field"}]}`)}
	plan := PlanRuntime(oldState, newState)
	if !plan.RequiresRestart {
		t.Fatalf("expected restart: %#v", plan)
	}
}

func TestPlanRuntimeNewPolicyLevelRequiresRestart(t *testing.T) {
	oldState := model.DesiredState{Inbounds: []model.ManagedInbound{testInbound(testUser("u1", "a", true))}}
	u2 := testUser("u2", "b", true)
	u2.Level = 7
	newState := model.DesiredState{Inbounds: []model.ManagedInbound{testInbound(testUser("u1", "a", true), u2)}}
	plan := PlanRuntime(oldState, newState)
	if !plan.RequiresRestart {
		t.Fatalf("expected restart for a new stats policy level: %#v", plan)
	}
}

func TestPlanRuntimeNewInboundWithNewPolicyLevelRequiresRestart(t *testing.T) {
	u := testUser("u9", "z", true)
	u.Level = 9
	newInbound := model.ManagedInbound{ID: "in-9", Tag: "vless-9443", Protocol: "vless", Port: 9443, Users: []model.ManagedUser{u}}
	plan := PlanRuntime(model.DesiredState{}, model.DesiredState{Inbounds: []model.ManagedInbound{newInbound}})
	if !plan.RequiresRestart {
		t.Fatalf("expected restart for new inbound policy level: %#v", plan)
	}
}
