package model

import "encoding/json"

type AgentConfig struct {
	NodeID      string `json:"node_id"`
	PanelURL    string `json:"panel_url"`
	PanelToken  string `json:"panel_token"`
	SyncSeconds int    `json:"sync_seconds"`
	XrayBinary  string `json:"xray_binary"`
	XrayConfig  string `json:"xray_config"`
	XrayAPI     string `json:"xray_api"`
	StateFile   string `json:"state_file"`
	Listen      string `json:"listen"`
}

type DesiredState struct {
	Version   string            `json:"version"`
	Enabled   bool              `json:"enabled"`
	Mode      string            `json:"mode,omitempty"`
	Inbounds  []ManagedInbound  `json:"inbounds"`
	Outbounds []json.RawMessage `json:"outbounds,omitempty"`
	Routing   json.RawMessage   `json:"routing,omitempty"`
	DNS       json.RawMessage   `json:"dns,omitempty"`
}

type ManagedInbound struct {
	ID             string                 `json:"id"`
	Tag            string                 `json:"tag"`
	Listen         string                 `json:"listen,omitempty"`
	Port           any                    `json:"port"`
	Protocol       string                 `json:"protocol"`
	Settings       map[string]any         `json:"settings,omitempty"`
	StreamSettings map[string]any         `json:"stream_settings,omitempty"`
	Sniffing       map[string]any         `json:"sniffing,omitempty"`
	Users          []ManagedUser          `json:"users,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type ManagedUser struct {
	ID         string         `json:"id"`
	Enabled    bool           `json:"enabled"`
	Email      string         `json:"email,omitempty"`
	Level      int            `json:"level,omitempty"`
	Credential map[string]any `json:"credential"`
	Limits     UserLimits     `json:"limits,omitempty"`
}

type UserLimits struct {
	TrafficBytes    int64 `json:"traffic_bytes,omitempty"`
	UploadBPS       int64 `json:"upload_bps,omitempty"`
	DownloadBPS     int64 `json:"download_bps,omitempty"`
	IPLimit         int   `json:"ip_limit,omitempty"`
	DeviceLimit     int   `json:"device_limit,omitempty"`
	ConnectionLimit int   `json:"connection_limit,omitempty"`
	ExpiresAt       int64 `json:"expires_at,omitempty"`
}

type TrafficRecord struct {
	Name      string `json:"name"`
	Value     int64  `json:"value"`
	UserID    string `json:"user_id,omitempty"`
	InboundID string `json:"inbound_id,omitempty"`
	Direction string `json:"direction,omitempty"`
}

type Heartbeat struct {
	NodeID       string  `json:"node_id"`
	AgentVersion string  `json:"agent_version"`
	XrayVersion  string  `json:"xray_version,omitempty"`
	Healthy      bool    `json:"healthy"`
	XrayRunning  bool    `json:"xray_running"`
	CPUPercent   float64 `json:"cpu_percent,omitempty"`
	MemoryBytes  uint64  `json:"memory_bytes,omitempty"`
	Load1        float64 `json:"load1,omitempty"`
	StateVersion string  `json:"state_version,omitempty"`
	Message      string  `json:"message,omitempty"`
}
