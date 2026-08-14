package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"xnode-agent/internal/health"
	"xnode-agent/internal/limits"
	"xnode-agent/internal/model"
	"xnode-agent/internal/panel"
	"xnode-agent/internal/xray"
)

const Version = "0.1.0"

type Agent struct {
	Cfg          model.AgentConfig
	Panel        *panel.Client
	Xray         *xray.Manager
	Limiter      limits.Backend
	stateVersion string
}

func New(cfg model.AgentConfig) *Agent {
	return &Agent{Cfg: cfg, Panel: panel.New(cfg.PanelURL, cfg.PanelToken), Xray: &xray.Manager{Binary: cfg.XrayBinary, Config: cfg.XrayConfig, API: cfg.XrayAPI}, Limiter: limits.ObserveOnly{}}
}

func (a *Agent) Run(ctx context.Context) error {
	if err := a.sync(ctx); err != nil {
		log.Printf("initial sync: %v", err)
	}
	t := time.NewTicker(time.Duration(a.Cfg.SyncSeconds) * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = a.Xray.Stop()
			return ctx.Err()
		case <-t.C:
			if err := a.sync(ctx); err != nil {
				log.Printf("sync: %v", err)
			}
		}
	}
}

func (a *Agent) sync(ctx context.Context) error {
	state, err := a.Panel.DesiredState(ctx, a.Cfg.NodeID)
	if err != nil {
		a.sendHeartbeat(ctx, false, err.Error())
		return err
	}
	if !state.Enabled || state.Mode == "disabled" || state.Mode == "maintenance" {
		_ = a.Xray.Stop()
		a.stateVersion = state.Version
		a.persistState(state)
		a.sendHeartbeat(ctx, true, "node disabled/maintenance")
		return nil
	}
	cfg, err := xray.BuildConfig(state, a.Cfg.XrayAPI)
	if err != nil {
		a.sendHeartbeat(ctx, false, err.Error())
		return err
	}
	changed, err := a.Xray.Apply(ctx, cfg)
	if err != nil {
		a.sendHeartbeat(ctx, false, err.Error())
		return err
	}
	if !changed && !a.Xray.Running() {
		if err := a.Xray.Start(); err != nil {
			return err
		}
	}
	if err := a.applyLimits(ctx, state); err != nil {
		log.Printf("limits: %v", err)
	}
	if records, err := a.Xray.QueryStats(ctx, true); err == nil && len(records) > 0 {
		if err := a.Panel.Traffic(ctx, a.Cfg.NodeID, records); err != nil {
			log.Printf("traffic report: %v", err)
		}
	}
	a.stateVersion = state.Version
	_ = a.persistState(state)
	a.sendHeartbeat(ctx, true, "ok")
	return nil
}

func (a *Agent) applyLimits(ctx context.Context, state model.DesiredState) error {
	var errs []error
	for _, in := range state.Inbounds {
		for _, u := range in.Users {
			if !u.Enabled {
				continue
			}
			if err := a.Limiter.ApplyUser(ctx, a.Cfg.NodeID, in.ID, u.ID, u.Limits); err != nil {
				errs = append(errs, fmt.Errorf("%s/%s: %w", in.ID, u.ID, err))
			}
		}
	}
	return errors.Join(errs...)
}

func (a *Agent) sendHeartbeat(ctx context.Context, healthy bool, msg string) {
	mem, load := health.Snapshot()
	hb := model.Heartbeat{NodeID: a.Cfg.NodeID, AgentVersion: Version, XrayVersion: a.Xray.Version(ctx), Healthy: healthy, XrayRunning: a.Xray.Running(), MemoryBytes: mem, Load1: load, StateVersion: a.stateVersion, Message: msg}
	if err := a.Panel.Heartbeat(ctx, hb); err != nil {
		log.Printf("heartbeat: %v", err)
	}
}

func (a *Agent) persistState(state model.DesiredState) error {
	if err := os.MkdirAll(filepath.Dir(a.Cfg.StateFile), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(state, "", "  ")
	return os.WriteFile(a.Cfg.StateFile, b, 0o600)
}
