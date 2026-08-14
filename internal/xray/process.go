package xray

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Manager struct {
	Binary     string
	Config     string
	API        string
	LimitsFile string
	mu         sync.Mutex
	cmd        *exec.Cmd
}

func (m *Manager) backupPath() string { return m.Config + ".last-good" }

func (m *Manager) Validate(ctx context.Context, candidate string) error {
	cmd := exec.CommandContext(ctx, m.Binary, "run", "-test", "-c", candidate)
	cmd.Env = m.commandEnv()
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xray config validation failed: %w: %s", err, out.String())
	}
	return nil
}

func (m *Manager) ValidateContent(ctx context.Context, content []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.validateContentLocked(ctx, content)
}

func (m *Manager) Store(ctx context.Context, content []byte) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, err := os.ReadFile(m.Config); err == nil && hash(old) == hash(content) {
		return false, nil
	}
	if err := m.validateContentLocked(ctx, content); err != nil {
		return false, err
	}
	if err := m.writeConfigLocked(content); err != nil {
		return false, err
	}
	// A successful runtime reconcile means this persisted config is known-good.
	_ = m.writeBackupLocked(content)
	return true, nil
}

func (m *Manager) Apply(ctx context.Context, content []byte) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, err := os.ReadFile(m.Config); err == nil && hash(old) == hash(content) {
		return false, nil
	}
	if err := m.validateContentLocked(ctx, content); err != nil {
		return false, err
	}
	old, _ := os.ReadFile(m.Config)
	if len(old) > 0 {
		_ = m.writeBackupLocked(old)
	}
	if err := m.writeConfigLocked(content); err != nil {
		return false, err
	}
	if err := m.restartLocked(); err != nil {
		return false, m.rollbackLocked(err)
	}
	_ = m.writeBackupLocked(content)
	return true, nil
}

func (m *Manager) ForceApply(ctx context.Context, content []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateContentLocked(ctx, content); err != nil {
		return err
	}
	old, _ := os.ReadFile(m.Config)
	if len(old) > 0 && hash(old) != hash(content) {
		_ = m.writeBackupLocked(old)
	}
	if err := m.writeConfigLocked(content); err != nil {
		return err
	}
	if err := m.restartLocked(); err != nil {
		return m.rollbackLocked(err)
	}
	_ = m.writeBackupLocked(content)
	return nil
}

func (m *Manager) rollbackLocked(cause error) error {
	backup, err := os.ReadFile(m.backupPath())
	if err != nil || len(backup) == 0 {
		return cause
	}
	if err := m.writeConfigLocked(backup); err != nil {
		return fmt.Errorf("new config failed (%v); rollback write failed: %w", cause, err)
	}
	if err := m.restartLocked(); err != nil {
		return fmt.Errorf("new config failed (%v); rollback start failed: %w", cause, err)
	}
	return fmt.Errorf("new config failed and was rolled back to last-good: %w", cause)
}

func (m *Manager) validateContentLocked(ctx context.Context, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(m.Config), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(m.Config), ".xnode-candidate-*.json")
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
	return m.Validate(ctx, name)
}

func (m *Manager) writeConfigLocked(content []byte) error {
	return atomicWrite(m.Config, content, 0o600)
}

func (m *Manager) writeBackupLocked(content []byte) error {
	return atomicWrite(m.backupPath(), content, 0o600)
}

func atomicWrite(path string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".xnode-write-*")
	if err != nil {
		return err
	}
	name := f.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(name)
		}
	}()
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(content); err != nil {
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
	if err := os.Rename(name, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.isRunningLocked() {
		return nil
	}
	return m.startLocked()
}
func (m *Manager) Restart() error { m.mu.Lock(); defer m.mu.Unlock(); return m.restartLocked() }
func (m *Manager) Stop() error    { m.mu.Lock(); defer m.mu.Unlock(); return m.stopLocked() }
func (m *Manager) Running() bool  { m.mu.Lock(); defer m.mu.Unlock(); return m.isRunningLocked() }

func (m *Manager) startLocked() error {
	if m.LimitsFile != "" {
		_ = os.Remove(m.LimitsFile + ".core-ready")
	}
	cmd := exec.Command(m.Binary, "run", "-c", m.Config)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = m.commandEnv()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	m.cmd = cmd
	go func(c *exec.Cmd) { _ = c.Wait() }(cmd)
	// Detect immediate crashes/config/runtime failures instead of reporting a
	// successful recovery just because fork/exec succeeded.
	time.Sleep(350 * time.Millisecond)
	if !m.isRunningLocked() {
		return fmt.Errorf("xray exited during startup grace period")
	}
	return nil
}

func (m *Manager) commandEnv() []string {
	env := os.Environ()
	if m.LimitsFile != "" {
		env = append(env, "XNODE_LIMITS_FILE="+m.LimitsFile)
	}
	return env
}

func (m *Manager) stopLocked() error {
	if !m.isRunningLocked() {
		m.cmd = nil
		return nil
	}
	_ = syscall.Kill(-m.cmd.Process.Pid, syscall.SIGTERM)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !m.isRunningLocked() {
			m.cmd = nil
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = syscall.Kill(-m.cmd.Process.Pid, syscall.SIGKILL)
	m.cmd = nil
	return nil
}
func (m *Manager) restartLocked() error { _ = m.stopLocked(); return m.startLocked() }
func (m *Manager) isRunningLocked() bool {
	if m.cmd == nil || m.cmd.Process == nil {
		return false
	}
	return m.cmd.Process.Signal(syscall.Signal(0)) == nil
}

func (m *Manager) Recover() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.restartLocked(); err == nil {
		return nil
	}
	backup, err := os.ReadFile(m.backupPath())
	if err != nil || len(backup) == 0 {
		return fmt.Errorf("xray recovery failed and no last-good config is available")
	}
	if err := m.writeConfigLocked(backup); err != nil {
		return err
	}
	return m.restartLocked()
}

func (m *Manager) CorePatchReady() bool {
	if m.LimitsFile == "" {
		return false
	}
	_, err := os.Stat(m.LimitsFile + ".core-ready")
	return err == nil
}

func (m *Manager) APIHealthy(ctx context.Context) bool {
	if !m.Running() {
		return false
	}
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cctx, m.Binary, "api", "statsquery", "--server="+m.API).CombinedOutput()
	return err == nil && len(out) > 0
}

func (m *Manager) Version(ctx context.Context) string {
	out, err := exec.CommandContext(ctx, m.Binary, "version").CombinedOutput()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(out))
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	return line
}
func hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
