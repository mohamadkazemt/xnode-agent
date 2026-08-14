package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"xnode-agent/internal/admin"
	"xnode-agent/internal/health"
	"xnode-agent/internal/limits"
	"xnode-agent/internal/model"
	"xnode-agent/internal/panel"
	"xnode-agent/internal/policy"
	"xnode-agent/internal/session"
	"xnode-agent/internal/traffic"
	"xnode-agent/internal/xray"
)

const Version = "1.0.0"

type Agent struct {
	Cfg      model.AgentConfig
	Panel    *panel.Client
	Xray     *xray.Manager
	Limiter  limits.Backend
	Sessions *session.Tracker
	Spool    traffic.Spool
	Metrics  *health.Sampler

	stateVersion string
	onlineUsers  int
	trackedIPs   int
	mode         string
	drainReady   bool
	strictReady  bool
	region       string
	group        string
	tags         []string
	weight       int

	statusMu sync.RWMutex
	lastHB   model.Heartbeat
}

func New(cfg model.AgentConfig) *Agent {
	var backend limits.Backend
	if cfg.StrictLimitBackendURL != "" {
		backend = limits.New(cfg.StrictLimitBackendURL)
	}
	return &Agent{
		Cfg:      cfg,
		Panel:    panel.New(cfg.PanelURL, cfg.PanelToken),
		Xray:     &xray.Manager{Binary: cfg.XrayBinary, Config: cfg.XrayConfig, API: cfg.XrayAPI, LimitsFile: cfg.XrayLimitsFile},
		Limiter:  backend,
		Sessions: session.NewTracker(),
		Spool:    traffic.Spool{Dir: cfg.TrafficSpoolDir},
		Metrics:  health.NewSampler(cfg.NetworkInterface),
		mode:     "starting", region: cfg.Region, group: cfg.Group, tags: append([]string(nil), cfg.Tags...), weight: cfg.Weight,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	if err := a.preparePaths(); err != nil {
		return err
	}
	adminErr := make(chan error, 1)
	go func() { adminErr <- admin.Serve(ctx, a.Cfg.Listen, a.Status) }()

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
		case err := <-adminErr:
			if err != nil {
				return fmt.Errorf("admin server: %w", err)
			}
			return nil
		case <-t.C:
			if err := a.sync(ctx); err != nil {
				log.Printf("sync: %v", err)
			}
		}
	}
}

func (a *Agent) Status() model.Heartbeat {
	a.statusMu.RLock()
	defer a.statusMu.RUnlock()
	hb := a.lastHB
	hb.Tags = append([]string(nil), hb.Tags...)
	return hb
}

func (a *Agent) preparePaths() error {
	dirs := []string{filepath.Dir(a.Cfg.XrayConfig), filepath.Dir(a.Cfg.XrayAccessLog), filepath.Dir(a.Cfg.XrayLimitsFile), filepath.Dir(a.Cfg.StateFile), a.Cfg.TrafficSpoolDir}
	for _, dir := range dirs {
		if dir == "" || dir == "." {
			continue
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func (a *Agent) sync(ctx context.Context) error {
	previous, havePrevious := a.loadState()

	a.retryTraffic(ctx)
	if err := a.collectTraffic(ctx); err != nil {
		log.Printf("traffic collection: %v", err)
	}

	now := time.Now()
	policySessions, reportSessions := a.sessionSnapshot(ctx, now)
	a.onlineUsers, a.trackedIPs = session.Counts(policySessions)

	pendingUsage, err := a.Spool.PendingUsage()
	if err != nil {
		log.Printf("traffic spool usage: %v", err)
		pendingUsage = map[policy.UsageKey]int64{}
	}

	desired, err := a.Panel.DesiredState(ctx, a.Cfg.NodeID)
	if err != nil {
		a.sendHeartbeat(ctx, false, "panel desired-state: "+err.Error(), false)
		return err
	}
	if err := validateMode(desired.Mode); err != nil {
		a.sendHeartbeat(ctx, false, err.Error(), false)
		return err
	}

	thresholdTriggered := false
	if desired.Node.TrafficThresholdBytes > 0 && desired.Node.TrafficUsedBytes >= desired.Node.TrafficThresholdBytes && desired.Mode != "disabled" && desired.Mode != "maintenance" {
		desired.Mode = "draining"
		thresholdTriggered = true
	}
	if desired.Mode == "draining" && havePrevious {
		desired = policy.ApplyDrain(previous, desired)
	}

	eval := policy.Evaluate(desired, policySessions, pendingUsage, now)
	state := eval.State
	a.setNodeMetadata(state)
	target := state.Node.DrainTargetOnline
	if target < 0 {
		target = 0
	}
	a.drainReady = state.Mode == "draining" && a.onlineUsers <= target

	if err := limits.WriteCorePolicy(a.Cfg.XrayLimitsFile, state, now); err != nil {
		a.sendHeartbeat(ctx, false, "write strict limits: "+err.Error(), false)
		return err
	}
	strictUsers := limits.StrictPolicyEmailsFromFile(a.Cfg.XrayLimitsFile, now)

	if !state.Enabled || state.Mode == "disabled" || state.Mode == "maintenance" {
		_ = a.Xray.Stop()
		a.strictReady = len(strictUsers) == 0
		a.stateVersion = state.Version
		_ = a.persistState(state)
		a.reportSessions(ctx, reportSessions, eval.Violations, now)
		a.sendHeartbeat(ctx, true, "node "+state.Mode, false)
		return nil
	}

	cfg, err := xray.BuildConfig(state, a.Cfg.XrayAPI, a.Cfg.XrayAccessLog)
	if err != nil {
		a.sendHeartbeat(ctx, false, err.Error(), false)
		return err
	}

	if a.Xray.Running() && havePrevious && previous.Enabled && previous.Mode != "disabled" && previous.Mode != "maintenance" {
		plan := xray.PlanRuntime(previous, state)
		if !plan.RequiresRestart {
			if err := a.Xray.ValidateContent(ctx, cfg); err == nil {
				if err = a.Xray.ApplyRuntime(ctx, plan); err == nil {
					if _, err = a.Xray.Store(ctx, cfg); err == nil {
						if len(plan.Operations) > 0 {
							log.Printf("hot reload: %d operation(s)", len(plan.Operations))
						}
					} else {
						log.Printf("hot reload config persistence failed, restarting: %v", err)
					}
				} else {
					log.Printf("hot reload failed, restarting with desired config: %v", err)
				}
			} else {
				log.Printf("desired config validation failed before hot reload: %v", err)
			}
			if err != nil {
				if fallbackErr := a.Xray.ForceApply(ctx, cfg); fallbackErr != nil {
					a.sendHeartbeat(ctx, false, fallbackErr.Error(), false)
					return errors.Join(err, fallbackErr)
				}
			}
		} else {
			log.Printf("hot reload skipped: %s", plan.Reason)
			if err := a.Xray.ForceApply(ctx, cfg); err != nil {
				a.sendHeartbeat(ctx, false, err.Error(), false)
				return err
			}
		}
	} else {
		changed, err := a.Xray.Apply(ctx, cfg)
		if err != nil {
			a.sendHeartbeat(ctx, false, err.Error(), false)
			return err
		}
		if !changed && !a.Xray.Running() {
			if err := a.Xray.Start(); err != nil {
				a.sendHeartbeat(ctx, false, err.Error(), false)
				return err
			}
		}
	}

	if a.Limiter != nil {
		if err := a.applyExternalLimits(ctx, state); err != nil {
			log.Printf("external limits: %v", err)
		}
	}

	apiHealthy := a.Xray.APIHealthy(ctx)
	if a.Xray.Running() && !apiHealthy {
		log.Printf("xray API unhealthy; attempting recovery")
		if err := a.Xray.Recover(); err != nil {
			a.sendHeartbeat(ctx, false, "xray recovery failed: "+err.Error(), false)
			return err
		}
		apiHealthy = a.Xray.APIHealthy(ctx)
	}

	a.strictReady = len(strictUsers) == 0 || a.Xray.CorePatchReady()
	healthy := apiHealthy
	msg := "ok"
	if thresholdTriggered {
		msg = "traffic threshold reached; draining"
	}
	if len(eval.Violations) > 0 {
		msg = fmt.Sprintf("%s; %d policy block(s)", msg, len(eval.Violations))
	}
	if len(strictUsers) > 0 && !a.strictReady {
		msg += "; strict speed/connection core patch not detected"
		if a.Cfg.RequirePatchedCore {
			healthy = false
		}
	}
	if a.drainReady {
		msg += "; drain ready"
	}

	a.reportSessions(ctx, reportSessions, eval.Violations, now)
	a.stateVersion = state.Version
	_ = a.persistState(state)
	a.sendHeartbeat(ctx, healthy, msg, apiHealthy)
	return nil
}

func validateMode(mode string) error {
	switch mode {
	case "", "active", "draining", "maintenance", "disabled":
		return nil
	default:
		return fmt.Errorf("unsupported node mode %q", mode)
	}
}

func (a *Agent) sessionSnapshot(ctx context.Context, now time.Time) ([]model.SessionRecord, []model.SessionRecord) {
	if err := a.Sessions.ConsumeFile(a.Cfg.XrayAccessLog, now); err != nil {
		log.Printf("access log: %v", err)
	}
	recent := a.Sessions.Records(now, time.Duration(a.Cfg.IPWindowSeconds)*time.Second)
	for i := range recent {
		recent[i].Source = "access-log"
	}
	if !a.Xray.Running() {
		return recent, recent
	}
	exact, err := a.Xray.QueryOnlineSessions(ctx)
	if err != nil {
		log.Printf("native online stats unavailable, using access-log fallback: %v", err)
		return recent, recent
	}
	return exact, mergeSessionReports(exact, recent)
}

func mergeSessionReports(exact, recent []model.SessionRecord) []model.SessionRecord {
	type key struct{ u, i string }
	m := map[key]model.SessionRecord{}
	for _, r := range recent {
		m[key{r.UserID, r.InboundID}] = r
	}
	out := make([]model.SessionRecord, 0, len(exact)+len(recent))
	seen := map[key]struct{}{}
	for _, e := range exact {
		k := key{e.UserID, e.InboundID}
		seen[k] = struct{}{}
		if r, ok := m[k]; ok {
			e.RecentConnections = r.RecentConnections
		}
		out = append(out, e)
	}
	for _, r := range recent {
		k := key{r.UserID, r.InboundID}
		if _, ok := seen[k]; !ok {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].InboundID == out[j].InboundID {
			return out[i].UserID < out[j].UserID
		}
		return out[i].InboundID < out[j].InboundID
	})
	return out
}

func (a *Agent) setNodeMetadata(state model.DesiredState) {
	a.mode = state.Mode
	if a.mode == "" {
		a.mode = "active"
	}
	a.region, a.group, a.tags, a.weight = a.Cfg.Region, a.Cfg.Group, append([]string(nil), a.Cfg.Tags...), a.Cfg.Weight
	if state.Node.Region != "" {
		a.region = state.Node.Region
	}
	if state.Node.Group != "" {
		a.group = state.Node.Group
	}
	if len(state.Node.Tags) > 0 {
		a.tags = append([]string(nil), state.Node.Tags...)
	}
	if state.Node.Weight > 0 {
		a.weight = state.Node.Weight
	}
}

func (a *Agent) retryTraffic(ctx context.Context) {
	batches, err := a.Spool.List()
	if err != nil {
		log.Printf("traffic spool list: %v", err)
		return
	}
	for _, batch := range batches {
		if err := a.Panel.TrafficBatch(ctx, batch); err != nil {
			log.Printf("traffic retry %s: %v", batch.EventID, err)
			return
		}
		if err := a.Spool.Delete(batch.EventID); err != nil {
			log.Printf("traffic spool delete %s: %v", batch.EventID, err)
			return
		}
	}
}

func (a *Agent) collectTraffic(ctx context.Context) error {
	if !a.Xray.Running() {
		return nil
	}
	records, err := a.Xray.QueryStats(ctx, true)
	if err != nil || len(records) == 0 {
		return err
	}
	batch := model.TrafficBatch{EventID: traffic.NewEventID(), NodeID: a.Cfg.NodeID, CollectedAt: time.Now().Unix(), Records: records}
	if err := a.Spool.Enqueue(batch); err != nil {
		return fmt.Errorf("spool traffic after counter reset: %w", err)
	}
	if err := a.Panel.TrafficBatch(ctx, batch); err != nil {
		log.Printf("traffic report queued %s: %v", batch.EventID, err)
		return nil
	}
	return a.Spool.Delete(batch.EventID)
}

func (a *Agent) applyExternalLimits(ctx context.Context, state model.DesiredState) error {
	var errs []error
	for _, in := range state.Inbounds {
		for _, u := range in.Users {
			if !u.Enabled {
				if err := a.Limiter.RemoveUser(ctx, a.Cfg.NodeID, in.ID, u.ID); err != nil {
					errs = append(errs, fmt.Errorf("remove %s/%s: %w", in.ID, u.ID, err))
				}
				continue
			}
			if err := a.Limiter.ApplyUser(ctx, a.Cfg.NodeID, in.ID, u.ID, u.Limits); err != nil {
				errs = append(errs, fmt.Errorf("%s/%s: %w", in.ID, u.ID, err))
			}
		}
	}
	return errors.Join(errs...)
}

func (a *Agent) reportSessions(ctx context.Context, records []model.SessionRecord, violations []model.PolicyViolation, now time.Time) {
	if !a.Cfg.ReportSessions {
		return
	}
	report := model.SessionReport{NodeID: a.Cfg.NodeID, GeneratedAt: now.Unix(), WindowSec: a.Cfg.IPWindowSeconds, Records: records, Violations: violations}
	if err := a.Panel.Sessions(ctx, report); err != nil {
		log.Printf("session report: %v", err)
	}
}

func (a *Agent) sendHeartbeat(ctx context.Context, healthy bool, msg string, apiHealthy bool) {
	m := a.Metrics.Snapshot(time.Now())
	hb := model.Heartbeat{
		NodeID: a.Cfg.NodeID, AgentVersion: Version, XrayVersion: a.Xray.Version(ctx), Healthy: healthy,
		XrayRunning: a.Xray.Running(), XrayAPI: apiHealthy, CPUPercent: m.CPUPercent, MemoryBytes: m.MemoryBytes, Load1: m.Load1,
		NetworkRX: m.NetworkRX, NetworkTX: m.NetworkTX, NetworkRXBPS: m.NetworkRXBPS, NetworkTXBPS: m.NetworkTXBPS,
		OnlineUsers: a.onlineUsers, TrackedIPs: a.trackedIPs, StateVersion: a.stateVersion, Mode: a.mode, DrainReady: a.drainReady,
		Region: a.region, Group: a.group, Tags: append([]string(nil), a.tags...), Weight: a.weight, StrictLimitsReady: a.strictReady, Message: msg,
	}
	a.statusMu.Lock()
	a.lastHB = hb
	a.statusMu.Unlock()
	if err := a.Panel.Heartbeat(ctx, hb); err != nil {
		log.Printf("heartbeat: %v", err)
	}
}

func (a *Agent) loadState() (model.DesiredState, bool) {
	var state model.DesiredState
	b, err := os.ReadFile(a.Cfg.StateFile)
	if err != nil {
		return state, false
	}
	if json.Unmarshal(b, &state) != nil {
		return model.DesiredState{}, false
	}
	return state, true
}
func (a *Agent) persistState(state model.DesiredState) error {
	dir := filepath.Dir(a.Cfg.StateFile)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".xnode-state-*.json")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, a.Cfg.StateFile)
}
