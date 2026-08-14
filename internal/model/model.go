package model

import "encoding/json"

type AgentConfig struct {
	NodeID                string   `json:"node_id"`
	PanelURL              string   `json:"panel_url"`
	PanelToken            string   `json:"panel_token"`
	SyncSeconds           int      `json:"sync_seconds"`
	XrayBinary            string   `json:"xray_binary"`
	XrayConfig            string   `json:"xray_config"`
	XrayAPI               string   `json:"xray_api"`
	XrayAccessLog         string   `json:"xray_access_log"`
	XrayLimitsFile        string   `json:"xray_limits_file"`
	StateFile             string   `json:"state_file"`
	TrafficSpoolDir       string   `json:"traffic_spool_dir"`
	IPWindowSeconds       int      `json:"ip_window_seconds"`
	ReportSessions        bool     `json:"report_sessions"`
	StrictLimitBackendURL string   `json:"strict_limit_backend_url,omitempty"`
	RequirePatchedCore    bool     `json:"require_patched_core,omitempty"`
	Listen                string   `json:"listen"`
	NetworkInterface      string   `json:"network_interface,omitempty"`
	Region                string   `json:"region,omitempty"`
	Group                 string   `json:"group,omitempty"`
	Tags                  []string `json:"tags,omitempty"`
	Weight                int      `json:"weight,omitempty"`
}

type DesiredState struct {
	Version   string            `json:"version"`
	Enabled   bool              `json:"enabled"`
	Mode      string            `json:"mode,omitempty"`
	Node      NodePolicy        `json:"node,omitempty"`
	Inbounds  []ManagedInbound  `json:"inbounds"`
	Outbounds []json.RawMessage `json:"outbounds,omitempty"`
	Routing   json.RawMessage   `json:"routing,omitempty"`
	DNS       json.RawMessage   `json:"dns,omitempty"`
}

type NodePolicy struct {
	Region                string   `json:"region,omitempty"`
	Group                 string   `json:"group,omitempty"`
	Tags                  []string `json:"tags,omitempty"`
	Weight                int      `json:"weight,omitempty"`
	TrafficThresholdBytes int64    `json:"traffic_threshold_bytes,omitempty"`
	TrafficUsedBytes      int64    `json:"traffic_used_bytes,omitempty"`
	DrainTargetOnline     int      `json:"drain_target_online,omitempty"`
}

type ManagedInbound struct {
	ID             string                 `json:"id"`
	Tag            string                 `json:"tag"`
	Listen         string                 `json:"listen,omitempty"`
	Port           any                    `json:"port"`
	Protocol       string                 `json:"protocol"`
	IPLimitMode    string                 `json:"ip_limit_mode,omitempty"`
	Settings       map[string]any         `json:"settings,omitempty"`
	StreamSettings map[string]any         `json:"stream_settings,omitempty"`
	Sniffing       map[string]any         `json:"sniffing,omitempty"`
	Users          []ManagedUser          `json:"users,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type ManagedUser struct {
	ID                string         `json:"id"`
	AccountID         string         `json:"account_id,omitempty"`
	Enabled           bool           `json:"enabled"`
	Email             string         `json:"email,omitempty"`
	OutboundTag       string         `json:"outbound_tag,omitempty"`
	SessionGeneration int64          `json:"session_generation,omitempty"`
	Level             int            `json:"level,omitempty"`
	Credential        map[string]any `json:"credential"`
	Limits            UserLimits     `json:"limits,omitempty"`
}

type UserLimits struct {
	TrafficBytes     int64 `json:"traffic_bytes,omitempty"`
	TrafficUsedBytes int64 `json:"traffic_used_bytes,omitempty"`
	UploadBPS        int64 `json:"upload_bps,omitempty"`
	DownloadBPS      int64 `json:"download_bps,omitempty"`
	IPLimit          int   `json:"ip_limit,omitempty"`
	DeviceLimit      int   `json:"device_limit,omitempty"`
	ConnectionLimit  int   `json:"connection_limit,omitempty"`
	ExpiresAt        int64 `json:"expires_at,omitempty"`
}

type TrafficRecord struct {
	Name      string `json:"name"`
	Value     int64  `json:"value"`
	UserID    string `json:"user_id,omitempty"`
	InboundID string `json:"inbound_id,omitempty"`
	Direction string `json:"direction,omitempty"`
}

type TrafficBatch struct {
	EventID     string          `json:"event_id"`
	NodeID      string          `json:"node_id"`
	CollectedAt int64           `json:"collected_at"`
	Records     []TrafficRecord `json:"records"`
}

type PolicyViolation struct {
	UserID    string `json:"user_id"`
	InboundID string `json:"inbound_id"`
	Reason    string `json:"reason"`
	Observed  int64  `json:"observed,omitempty"`
	Limit     int64  `json:"limit,omitempty"`
}

type SessionRecord struct {
	UserID            string   `json:"user_id"`
	InboundID         string   `json:"inbound_id"`
	IPs               []string `json:"ips"`
	LastSeen          int64    `json:"last_seen"`
	RecentConnections int      `json:"recent_connections,omitempty"`
	Source            string   `json:"source,omitempty"`
}

type SessionReport struct {
	NodeID      string            `json:"node_id"`
	GeneratedAt int64             `json:"generated_at"`
	WindowSec   int               `json:"window_sec"`
	Records     []SessionRecord   `json:"records"`
	Violations  []PolicyViolation `json:"violations,omitempty"`
}

type Heartbeat struct {
	NodeID            string   `json:"node_id"`
	AgentVersion      string   `json:"agent_version"`
	XrayVersion       string   `json:"xray_version,omitempty"`
	Healthy           bool     `json:"healthy"`
	XrayRunning       bool     `json:"xray_running"`
	XrayAPI           bool     `json:"xray_api_healthy,omitempty"`
	CPUPercent        float64  `json:"cpu_percent,omitempty"`
	MemoryBytes       uint64   `json:"memory_bytes,omitempty"`
	Load1             float64  `json:"load1,omitempty"`
	NetworkRX         uint64   `json:"network_rx_bytes,omitempty"`
	NetworkTX         uint64   `json:"network_tx_bytes,omitempty"`
	NetworkRXBPS      float64  `json:"network_rx_bps,omitempty"`
	NetworkTXBPS      float64  `json:"network_tx_bps,omitempty"`
	OnlineUsers       int      `json:"online_users,omitempty"`
	TrackedIPs        int      `json:"tracked_ips,omitempty"`
	StateVersion      string   `json:"state_version,omitempty"`
	Mode              string   `json:"mode,omitempty"`
	DrainReady        bool     `json:"drain_ready,omitempty"`
	Region            string   `json:"region,omitempty"`
	Group             string   `json:"group,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Weight            int      `json:"weight,omitempty"`
	StrictLimitsReady bool     `json:"strict_limits_ready,omitempty"`
	Message           string   `json:"message,omitempty"`
}
