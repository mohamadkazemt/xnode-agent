package config

import (
	"encoding/json"
	"fmt"
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
	if cfg.StateFile == "" {
		cfg.StateFile = "/var/lib/xnode/state.json"
	}
	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1:19090"
	}
	return cfg, nil
}
