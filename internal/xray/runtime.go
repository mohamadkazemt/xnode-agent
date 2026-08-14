package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"time"

	"xnode-agent/internal/model"
)

type RuntimeOpKind string

const (
	RuntimeAddInbound     RuntimeOpKind = "add_inbound"
	RuntimeRemoveInbound  RuntimeOpKind = "remove_inbound"
	RuntimeReplaceInbound RuntimeOpKind = "replace_inbound"
	RuntimeAddUser        RuntimeOpKind = "add_user"
	RuntimeRemoveUser     RuntimeOpKind = "remove_user"
)

type RuntimeOp struct {
	Kind      RuntimeOpKind
	OldTag    string
	Inbound   model.ManagedInbound
	User      model.ManagedUser
	UserEmail string
}

type RuntimePlan struct {
	Operations      []RuntimeOp
	RequiresRestart bool
	Reason          string
}

// PlanRuntime returns the minimum set of HandlerService-backed operations that
// can move a running Xray instance from previous to desired. Changes outside
// inbounds (outbounds/routing/dns) intentionally fall back to a full restart in
// v0.2 because they are not all safely hot-reloadable through one API surface.
func PlanRuntime(previous, desired model.DesiredState) RuntimePlan {
	if !runtimeGlobalEqual(previous, desired) {
		return RuntimePlan{RequiresRestart: true, Reason: "outbounds/routing/dns changed"}
	}

	previousLevels := activeLevels(previous)
	for _, in := range desired.Inbounds {
		for _, u := range in.Users {
			if !u.Enabled {
				continue
			}
			if _, ok := previousLevels[u.Level]; !ok {
				return RuntimePlan{RequiresRestart: true, Reason: fmt.Sprintf("new user policy level %d requires config reload", u.Level)}
			}
		}
	}
	prevByID := inboundByID(previous.Inbounds)
	nextByID := inboundByID(desired.Inbounds)
	plan := RuntimePlan{}

	for _, old := range previous.Inbounds {
		if _, ok := nextByID[old.ID]; !ok {
			plan.Operations = append(plan.Operations, RuntimeOp{Kind: RuntimeRemoveInbound, OldTag: old.Tag})
		}
	}

	for _, next := range desired.Inbounds {
		old, existed := prevByID[next.ID]
		if !existed {
			plan.Operations = append(plan.Operations, RuntimeOp{Kind: RuntimeAddInbound, Inbound: next})
			continue
		}

		if !inboundBaseEqual(old, next) {
			plan.Operations = append(plan.Operations, RuntimeOp{Kind: RuntimeReplaceInbound, OldTag: old.Tag, Inbound: next})
			continue
		}

		if usersRuntimeEqual(old.Users, next.Users) {
			continue
		}
		if !supportsHotUsers(next) {
			plan.Operations = append(plan.Operations, RuntimeOp{Kind: RuntimeReplaceInbound, OldTag: old.Tag, Inbound: next})
			continue
		}

		oldUsers := userByID(old.Users)
		newUsers := userByID(next.Users)

		for _, oldUser := range old.Users {
			newUser, ok := newUsers[oldUser.ID]
			if oldUser.Enabled && (!ok || !newUser.Enabled || !userRuntimeEqual(oldUser, newUser)) {
				plan.Operations = append(plan.Operations, RuntimeOp{
					Kind:      RuntimeRemoveUser,
					OldTag:    old.Tag,
					UserEmail: accountingEmail(oldUser.ID, old.ID),
				})
			}
		}

		for _, newUser := range next.Users {
			oldUser, ok := oldUsers[newUser.ID]
			if !newUser.Enabled {
				continue
			}
			if !ok || !oldUser.Enabled || !userRuntimeEqual(oldUser, newUser) {
				plan.Operations = append(plan.Operations, RuntimeOp{Kind: RuntimeAddUser, Inbound: next, User: newUser})
			}
		}
	}

	return plan
}

func runtimeGlobalEqual(a, b model.DesiredState) bool {
	return rawMessagesEqual(a.Outbounds, b.Outbounds) &&
		bytesJSONEqual(a.Routing, b.Routing) &&
		bytesJSONEqual(a.DNS, b.DNS)
}

func rawMessagesEqual(a, b []json.RawMessage) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bytesJSONEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func bytesJSONEqual(a, b []byte) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	var av, bv any
	if json.Unmarshal(a, &av) != nil || json.Unmarshal(b, &bv) != nil {
		return string(a) == string(b)
	}
	return reflect.DeepEqual(av, bv)
}

func inboundByID(inbounds []model.ManagedInbound) map[string]model.ManagedInbound {
	res := make(map[string]model.ManagedInbound, len(inbounds))
	for _, in := range inbounds {
		res[in.ID] = in
	}
	return res
}

func userByID(users []model.ManagedUser) map[string]model.ManagedUser {
	res := make(map[string]model.ManagedUser, len(users))
	for _, u := range users {
		res[u.ID] = u
	}
	return res
}

func inboundBaseEqual(a, b model.ManagedInbound) bool {
	aa, bb := a, b
	aa.Users, bb.Users = nil, nil
	aa.Metadata, bb.Metadata = nil, nil
	return reflect.DeepEqual(aa, bb)
}

func usersRuntimeEqual(a, b []model.ManagedUser) bool {
	if len(a) != len(b) {
		return false
	}
	am, bm := userByID(a), userByID(b)
	if len(am) != len(bm) {
		return false
	}
	for id, au := range am {
		bu, ok := bm[id]
		if !ok || !userRuntimeEqual(au, bu) || au.Enabled != bu.Enabled {
			return false
		}
	}
	return true
}

// userRuntimeEqual intentionally ignores panel-only fields (limits and Email).
// Xray's accounting email is generated from stable user/inbound IDs.
func userRuntimeEqual(a, b model.ManagedUser) bool {
	return a.Level == b.Level && reflect.DeepEqual(a.Credential, b.Credential)
}

func supportsHotUsers(in model.ManagedInbound) bool {
	switch strings.ToLower(in.Protocol) {
	case "vless", "vmess", "trojan", "shadowsocks":
		return true
	default:
		return false
	}
}

func activeLevels(state model.DesiredState) map[int]struct{} {
	levels := map[int]struct{}{0: {}}
	for _, in := range state.Inbounds {
		for _, u := range in.Users {
			if u.Enabled {
				levels[u.Level] = struct{}{}
			}
		}
	}
	return levels
}

// ApplyRuntime executes a hot-reload plan using Xray's own API CLI commands.
// The commands are thin clients for HandlerService (adi/rmi/adu/rmu).
func (m *Manager) ApplyRuntime(ctx context.Context, plan RuntimePlan) error {
	if plan.RequiresRestart {
		return fmt.Errorf("runtime plan requires restart: %s", plan.Reason)
	}
	for _, op := range plan.Operations {
		var err error
		switch op.Kind {
		case RuntimeAddInbound:
			err = m.addInboundRuntime(ctx, op.Inbound)
		case RuntimeRemoveInbound:
			err = m.removeInboundRuntime(ctx, op.OldTag)
		case RuntimeReplaceInbound:
			err = m.removeInboundRuntime(ctx, op.OldTag)
			if err == nil {
				// Xray has historically had a short race window when a tag is
				// removed and re-added immediately. A small delay makes replace
				// operations deterministic while keeping the node online.
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(250 * time.Millisecond):
				}
				err = m.addInboundRuntime(ctx, op.Inbound)
			}
		case RuntimeAddUser:
			err = m.addUserRuntime(ctx, op.Inbound, op.User)
		case RuntimeRemoveUser:
			err = m.removeUserRuntime(ctx, op.OldTag, op.UserEmail)
		default:
			err = fmt.Errorf("unknown runtime operation %q", op.Kind)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op.Kind, err)
		}
	}
	return nil
}

func (m *Manager) addInboundRuntime(ctx context.Context, in model.ManagedInbound) error {
	b, err := BuildInboundDocument(in)
	if err != nil {
		return err
	}
	return m.runAPIWithTempConfig(ctx, "adi", b)
}

func (m *Manager) removeInboundRuntime(ctx context.Context, tag string) error {
	return m.runAPI(ctx, "rmi", tag)
}

func (m *Manager) addUserRuntime(ctx context.Context, in model.ManagedInbound, user model.ManagedUser) error {
	b, err := BuildUserDocument(in, user)
	if err != nil {
		return err
	}
	return m.runAPIWithTempConfig(ctx, "adu", b)
}

func (m *Manager) removeUserRuntime(ctx context.Context, tag, email string) error {
	return m.runAPI(ctx, "rmu", "-tag="+tag, email)
}

func (m *Manager) runAPIWithTempConfig(ctx context.Context, action string, content []byte) error {
	f, err := os.CreateTemp("", "xnode-xray-api-*.json")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err := f.Write(content); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return m.runAPI(ctx, action, name)
}

func (m *Manager) runAPI(ctx context.Context, action string, args ...string) error {
	argv := []string{"api", action, "--server=" + m.API}
	argv = append(argv, args...)
	out, err := exec.CommandContext(ctx, m.Binary, argv...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("xray %s: %w: %s", action, err, strings.TrimSpace(string(out)))
	}
	return nil
}
