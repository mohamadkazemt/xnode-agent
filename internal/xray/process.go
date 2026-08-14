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

func (m *Manager) Apply(ctx context.Context, content []byte) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(m.Config), 0o755); err != nil {
		return false, err
	}
	if old, err := os.ReadFile(m.Config); err == nil && hash(old) == hash(content) {
		return false, nil
	}
	tmp := m.Config + ".candidate"
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		return false, err
	}
	if err := m.Validate(ctx, tmp); err != nil {
		_ = os.Remove(tmp)
		return false, err
	}
	if err := os.Rename(tmp, m.Config); err != nil {
		return false, err
	}
	if err := m.restartLocked(); err != nil {
		return false, err
	}
	return true, nil
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
