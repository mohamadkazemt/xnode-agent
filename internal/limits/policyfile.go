package limits

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"xnode-agent/internal/model"
	"xnode-agent/internal/xray"
)

type CorePolicyFile struct {
	Version     int                   `json:"version"`
	GeneratedAt int64                 `json:"generated_at"`
	Users       map[string]CorePolicy `json:"users"`
}

type CorePolicy struct {
	UploadBPS         int64 `json:"upload_bps,omitempty"`
	DownloadBPS       int64 `json:"download_bps,omitempty"`
	ConnectionLimit   int   `json:"connection_limit,omitempty"`
	IPLimit           int   `json:"ip_limit,omitempty"`
	Blocked           bool  `json:"blocked,omitempty"`
	TombstoneUntil    int64 `json:"tombstone_until,omitempty"`
	SessionGeneration int64 `json:"session_generation,omitempty"`
}

const removedUserTombstone = 5 * time.Minute

// WriteCorePolicy writes the data-path limits consumed by the maintained
// xnode Xray dispatcher patch. The file is always written atomically so a
// running core never observes partial JSON.
func WriteCorePolicy(path string, state model.DesiredState, now time.Time) error {
	if path == "" {
		return nil
	}
	doc := CorePolicyFile{Version: 1, GeneratedAt: now.Unix(), Users: map[string]CorePolicy{}}
	// Keep every current authenticated identity in the policy document, even
	// when it is unlimited. This lets a later desired-state deletion become a
	// temporary blocked tombstone so already-established sessions are closed.
	for _, in := range state.Inbounds {
		for _, u := range in.Users {
			ipLimit := u.Limits.IPLimit
			if strings.EqualFold(in.IPLimitMode, "off") {
				ipLimit = 0
			}
			doc.Users[xray.AccountingEmail(u.ID, in.ID)] = CorePolicy{
				UploadBPS: u.Limits.UploadBPS, DownloadBPS: u.Limits.DownloadBPS, ConnectionLimit: u.Limits.ConnectionLimit, IPLimit: ipLimit, Blocked: !u.Enabled, SessionGeneration: u.SessionGeneration,
			}
		}
	}

	// Preserve identities that disappeared from desired state as short-lived
	// blocked tombstones. HandlerService RemoveUser prevents new auth, while the
	// dispatcher policy makes existing sessions observe the removal on I/O.
	if b, err := os.ReadFile(path); err == nil {
		var old CorePolicyFile
		if json.Unmarshal(b, &old) == nil && old.Version == 1 {
			for email, p := range old.Users {
				if _, exists := doc.Users[email]; exists {
					continue
				}
				if p.TombstoneUntil == 0 {
					p.TombstoneUntil = now.Add(removedUserTombstone).Unix()
				}
				if p.TombstoneUntil > now.Unix() {
					p.Blocked = true
					p.UploadBPS, p.DownloadBPS, p.ConnectionLimit, p.IPLimit = 0, 0, 0, 0
					doc.Users[email] = p
				}
			}
		}
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".xnode-limits-*.json")
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
	return os.Rename(tmp, path)
}

func CorePolicyEmails(state model.DesiredState) []string {
	var out []string
	for _, in := range state.Inbounds {
		for _, u := range in.Users {
			if !u.Enabled || u.Limits.UploadBPS > 0 || u.Limits.DownloadBPS > 0 || u.Limits.ConnectionLimit > 0 || u.Limits.IPLimit > 0 {
				out = append(out, xray.AccountingEmail(u.ID, in.ID))
			}
		}
	}
	sort.Strings(out)
	return out
}

// StrictPolicyEmailsFromFile reports identities whose semantics require the
// maintained dispatcher patch (including removed-user tombstones).
func StrictPolicyEmailsFromFile(path string, now time.Time) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc CorePolicyFile
	if json.Unmarshal(b, &doc) != nil || doc.Version != 1 {
		return nil
	}
	out := make([]string, 0)
	for email, p := range doc.Users {
		if p.TombstoneUntil > 0 && p.TombstoneUntil <= now.Unix() {
			continue
		}
		if p.Blocked || p.UploadBPS > 0 || p.DownloadBPS > 0 || p.ConnectionLimit > 0 || p.IPLimit > 0 {
			out = append(out, email)
		}
	}
	sort.Strings(out)
	return out
}
