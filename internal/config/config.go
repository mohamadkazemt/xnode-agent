package config

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"

	"xnode-agent/internal/model"
)

func Load(path string) (model.AgentConfig, error) {
	var cfg model.AgentConfig
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if cfg.NodeID == "" || cfg.PanelURL == "" || cfg.PanelToken == "" {
		return cfg, fmt.Errorf("node_id, panel_url and panel_token are required")
	}
	u, err := url.Parse(cfg.PanelURL)
	if err != nil || u.Hostname() == "" {
		return cfg, fmt.Errorf("invalid panel_url")
	}
	if u.Scheme != "https" {
		host := u.Hostname()
		if u.Scheme != "http" || (host != "127.0.0.1" && host != "localhost" && host != "::1") {
			return cfg, fmt.Errorf("panel_url must use HTTPS unless it is loopback")
		}
	}
	if cfg.SyncSeconds <= 0 {
		cfg.SyncSeconds = 15
	}
	if cfg.XrayBinary == "" {
		cfg.XrayBinary = "/usr/local/bin/xray"
	}
	if cfg.XrayConfig == "" {
		cfg.XrayConfig = "/etc/xnode/xray.json"
	}
	if cfg.XrayAPI == "" {
		cfg.XrayAPI = "127.0.0.1:10085"
	}
	host, _, err := net.SplitHostPort(cfg.XrayAPI)
	if err != nil || (host != "127.0.0.1" && host != "localhost" && host != "::1") {
		return cfg, fmt.Errorf("xray_api must listen on loopback")
	}
	if cfg.XrayAccessLog == "" {
		cfg.XrayAccessLog = "/var/log/xnode/xray-access.log"
	}
	if cfg.XrayLimitsFile == "" {
		cfg.XrayLimitsFile = "/var/lib/xnode/limits.json"
	}
	if cfg.StateFile == "" {
		cfg.StateFile = "/var/lib/xnode/state.json"
	}
	if cfg.TrafficSpoolDir == "" {
		cfg.TrafficSpoolDir = "/var/lib/xnode/traffic-spool"
	}
	if cfg.IPWindowSeconds <= 0 {
		cfg.IPWindowSeconds = 120
	}
	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1:19090"
	}
	listenHost, _, err := net.SplitHostPort(cfg.Listen)
	if err != nil || (listenHost != "127.0.0.1" && listenHost != "localhost" && listenHost != "::1") {
		return cfg, fmt.Errorf("listen must be loopback")
	}
	if cfg.Weight <= 0 {
		cfg.Weight = 100
	}
	return cfg, nil
}
