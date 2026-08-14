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
	Binary string
	Config string
	API    string
	mu     sync.Mutex
	cmd    *exec.Cmd
}

func (m *Manager) Validate(ctx context.Context, candidate string) error {
	cmd := exec.CommandContext(ctx, m.Binary, "run", "-test", "-c", candidate)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xray config validation failed: %w: %s", err, out.String())
	}
	return nil
}

// ValidateContent validates a complete desired Xray config without replacing
// the active config file or restarting the process.
func (m *Manager) ValidateContent(ctx context.Context, content []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.validateContentLocked(ctx, content)
}

// Store validates and atomically persists a config without restarting Xray.
// It is used after a successful HandlerService hot reload so a later restart
// comes back with exactly the same desired state.
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
	if err := m.writeConfigLocked(content); err != nil {
		return false, err
	}
	if err := m.restartLocked(); err != nil {
		return false, err
	}
	return true, nil
}

// ForceApply validates, persists, and restarts even when the file on disk
// already matches. It is the safety fallback when a partial hot reload fails.
func (m *Manager) ForceApply(ctx context.Context, content []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateContentLocked(ctx, content); err != nil {
		return err
	}
	if err := m.writeConfigLocked(content); err != nil {
		return err
	}
	return m.restartLocked()
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
	if err := os.MkdirAll(filepath.Dir(m.Config), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(m.Config), ".xnode-config-*.json")
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
	if err := f.Chmod(0o600); err != nil {
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
	if err := os.Rename(name, m.Config); err != nil {
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
	cmd := exec.Command(m.Binary, "run", "-c", m.Config)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	m.cmd = cmd
	go func(c *exec.Cmd) { _ = c.Wait() }(cmd)
	return nil
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
	err := m.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
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
