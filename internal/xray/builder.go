package xray

import (
	"encoding/json"
	"fmt"
	"strings"

	"xnode-agent/internal/model"
)

func accountingEmail(userID, inboundID string) string {
	r := strings.NewReplacer(">", "_", "|", "_", " ", "_")
	return "u:" + r.Replace(userID) + "|i:" + r.Replace(inboundID)
}

func BuildConfig(state model.DesiredState, apiListen string) ([]byte, error) {
	cfg := map[string]any{
		"log": map[string]any{"loglevel": "warning"},
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
	if len(state.Routing) > 0 {
		var v any
		if err := json.Unmarshal(state.Routing, &v); err != nil {
			return nil, fmt.Errorf("routing: %w", err)
		}
		cfg["routing"] = v
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
	levels := map[string]any{"0": map[string]any{"statsUserUplink": true, "statsUserDownlink": true}}
	for _, in := range state.Inbounds {
		for _, u := range in.Users {
			if !u.Enabled {
				continue
			}
			levels[fmt.Sprintf("%d", u.Level)] = map[string]any{"statsUserUplink": true, "statsUserDownlink": true}
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
			client["email"] = accountingEmail(u.ID, in.ID)
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
		// Generic protocols are passed through as-is.
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

func ParseAccountingName(name string) (userID, inboundID, direction string, ok bool) {
	parts := strings.Split(name, ">>>")
	if len(parts) != 4 || parts[0] != "user" || parts[2] != "traffic" {
		return "", "", "", false
	}
	meta := parts[1]
	if !strings.HasPrefix(meta, "u:") {
		return "", "", "", false
	}
	x := strings.SplitN(strings.TrimPrefix(meta, "u:"), "|i:", 2)
	if len(x) != 2 {
		return "", "", "", false
	}
	return x[0], x[1], parts[3], true
}
