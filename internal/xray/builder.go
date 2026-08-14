package xray

import (
	"encoding/json"
	"fmt"
	"strings"

	"xnode-agent/internal/model"
)

func AccountingEmail(userID, inboundID string) string {
	r := strings.NewReplacer(">", "_", "|", "_", " ", "_")
	return "u:" + r.Replace(userID) + "|i:" + r.Replace(inboundID)
}

func BuildConfig(state model.DesiredState, apiListen string, accessLog ...string) ([]byte, error) {
	logCfg := map[string]any{"loglevel": "warning"}
	if len(accessLog) > 0 && accessLog[0] != "" {
		logCfg["access"] = accessLog[0]
	}
	cfg := map[string]any{
		"log": logCfg,
		"api": map[string]any{
			"tag":      "api",
			"listen":   apiListen,
			"services": []string{"HandlerService", "StatsService", "RoutingService", "LoggerService"},
		},
		"stats": map[string]any{},
		"policy": map[string]any{
			"levels": statsLevels(state),
			"system": map[string]any{"statsInboundUplink": true, "statsInboundDownlink": true, "statsOutboundUplink": true, "statsOutboundDownlink": true},
		},
	}

	inbounds := make([]any, 0, len(state.Inbounds))
	for _, in := range state.Inbounds {
		obj, err := buildInbound(in)
		if err != nil {
			return nil, fmt.Errorf("inbound %s: %w", in.ID, err)
		}
		inbounds = append(inbounds, obj)
	}
	cfg["inbounds"] = inbounds

	if len(state.Outbounds) == 0 {
		cfg["outbounds"] = []any{map[string]any{"tag": "direct", "protocol": "freedom"}, map[string]any{"tag": "block", "protocol": "blackhole"}}
	} else {
		var outs []any
		for _, raw := range state.Outbounds {
			var v any
			if err := json.Unmarshal(raw, &v); err != nil {
				return nil, fmt.Errorf("outbound: %w", err)
			}
			outs = append(outs, v)
		}
		cfg["outbounds"] = outs
	}
	routing, hasRouting, err := buildRouting(state)
	if err != nil {
		return nil, err
	}
	if hasRouting {
		cfg["routing"] = routing
	}
	if len(state.DNS) > 0 {
		var v any
		if err := json.Unmarshal(state.DNS, &v); err != nil {
			return nil, fmt.Errorf("dns: %w", err)
		}
		cfg["dns"] = v
	}
	return json.MarshalIndent(cfg, "", "  ")
}

func statsLevels(state model.DesiredState) map[string]any {
	levels := map[string]any{"0": map[string]any{"statsUserUplink": true, "statsUserDownlink": true, "statsUserOnline": true}}
	for _, in := range state.Inbounds {
		for _, u := range in.Users {
			if !u.Enabled {
				continue
			}
			levels[fmt.Sprintf("%d", u.Level)] = map[string]any{"statsUserUplink": true, "statsUserDownlink": true, "statsUserOnline": true}
		}
	}
	return levels
}

func buildInbound(in model.ManagedInbound) (map[string]any, error) {
	if in.ID == "" || in.Protocol == "" || in.Tag == "" {
		return nil, fmt.Errorf("id, tag and protocol are required")
	}
	obj := map[string]any{"tag": in.Tag, "protocol": in.Protocol, "port": in.Port}
	if in.Listen != "" {
		obj["listen"] = in.Listen
	}
	settings := cloneMap(in.Settings)
	p := strings.ToLower(in.Protocol)

	active := make([]model.ManagedUser, 0, len(in.Users))
	for _, u := range in.Users {
		if u.Enabled {
			active = append(active, u)
		}
	}

	switch p {
	case "vless", "vmess", "trojan", "shadowsocks":
		clients := make([]any, 0, len(active))
		for _, u := range active {
			client := cloneMap(u.Credential)
			// Always use a deterministic per-user/per-inbound email so Xray's
			// native user counters can be attributed without ambiguity.
			client["email"] = AccountingEmail(u.ID, in.ID)
			client["level"] = u.Level
			clients = append(clients, client)
		}
		settings["clients"] = clients
	case "wireguard":
		if len(active) > 0 {
			peers := make([]any, 0, len(active))
			for _, u := range active {
				peers = append(peers, cloneMap(u.Credential))
			}
			settings["peers"] = peers
		}
	default:
		// Generic protocols are passed through as-is. This keeps the agent compatible
		// with new Xray protocols without waiting for an agent release.
	}
	obj["settings"] = settings
	if in.StreamSettings != nil {
		obj["streamSettings"] = in.StreamSettings
	}
	if in.Sniffing != nil {
		obj["sniffing"] = in.Sniffing
	}
	return obj, nil
}

func buildRouting(state model.DesiredState) (map[string]any, bool, error) {
	base := map[string]any{}
	if len(state.Routing) > 0 {
		if err := json.Unmarshal(state.Routing, &base); err != nil {
			return nil, false, fmt.Errorf("routing: %w", err)
		}
	}
	var existing []any
	if raw, ok := base["rules"].([]any); ok {
		existing = raw
	}
	managed := make([]any, 0)
	for _, in := range state.Inbounds {
		for _, u := range in.Users {
			if !u.Enabled || u.OutboundTag == "" {
				continue
			}
			managed = append(managed, map[string]any{
				"type":        "field",
				"ruleTag":     "xnode-user-" + sanitizeTag(in.ID) + "-" + sanitizeTag(u.ID),
				"user":        []string{AccountingEmail(u.ID, in.ID)},
				"outboundTag": u.OutboundTag,
			})
		}
	}
	if len(managed) == 0 && len(base) == 0 {
		return nil, false, nil
	}
	if len(managed) > 0 {
		base["rules"] = append(managed, existing...)
	}
	return base, true, nil
}

func sanitizeTag(s string) string {
	r := strings.NewReplacer(" ", "_", ">", "_", "|", "_", "/", "_")
	return r.Replace(s)
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func ParseAccountingEmail(email string) (userID, inboundID string, ok bool) {
	if !strings.HasPrefix(email, "u:") {
		return "", "", false
	}
	x := strings.SplitN(strings.TrimPrefix(email, "u:"), "|i:", 2)
	if len(x) != 2 || x[0] == "" || x[1] == "" {
		return "", "", false
	}
	return x[0], x[1], true
}

func ParseAccountingName(name string) (userID, inboundID, direction string, ok bool) {
	parts := strings.Split(name, ">>>")
	if len(parts) != 4 || parts[0] != "user" || parts[2] != "traffic" {
		return "", "", "", false
	}
	userID, inboundID, ok = ParseAccountingEmail(parts[1])
	if !ok {
		return "", "", "", false
	}
	return userID, inboundID, parts[3], true
}

// BuildInboundDocument returns a minimal Xray JSON document accepted by
// `xray api adi`. It intentionally uses the same builder as the persisted
// configuration so hot-reloaded and restart-loaded inbounds stay identical.
func BuildInboundDocument(in model.ManagedInbound) ([]byte, error) {
	obj, err := buildInbound(in)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(map[string]any{"inbounds": []any{obj}}, "", "  ")
}

// BuildUserDocument returns a one-user inbound document accepted by
// `xray api adu`. The Xray CLI builds the inbound config and extracts its user
// object before calling HandlerService.AlterInbound(AddUserOperation).
func BuildUserDocument(in model.ManagedInbound, user model.ManagedUser) ([]byte, error) {
	one := in
	user.Enabled = true
	one.Users = []model.ManagedUser{user}
	return BuildInboundDocument(one)
}
