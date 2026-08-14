package traffic

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"xnode-agent/internal/model"
	"xnode-agent/internal/policy"
)

type Spool struct{ Dir string }

func NewEventID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func (s Spool) Enqueue(batch model.TrafficBatch) error {
	if batch.EventID == "" {
		return fmt.Errorf("event_id is required")
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.Dir, batch.EventID+".json.tmp")
	final := filepath.Join(s.Dir, batch.EventID+".json")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, final)
}

func (s Spool) List() ([]model.TrafficBatch, error) {
	ents, err := os.ReadDir(s.Dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(ents))
	for _, e := range ents {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, e.Name())
		}
	}
	out := make([]model.TrafficBatch, 0, len(files))
	for _, name := range files {
		b, err := os.ReadFile(filepath.Join(s.Dir, name))
		if err != nil {
			return nil, err
		}
		var batch model.TrafficBatch
		if err := json.Unmarshal(b, &batch); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		out = append(out, batch)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CollectedAt == out[j].CollectedAt {
			return out[i].EventID < out[j].EventID
		}
		return out[i].CollectedAt < out[j].CollectedAt
	})
	return out, nil
}

func (s Spool) Delete(eventID string) error {
	err := os.Remove(filepath.Join(s.Dir, eventID+".json"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s Spool) PendingUsage() (map[policy.UsageKey]int64, error) {
	batches, err := s.List()
	if err != nil {
		return nil, err
	}
	out := map[policy.UsageKey]int64{}
	for _, b := range batches {
		policy.MergeUsage(out, policy.PendingUsage(b.Records))
	}
	return out, nil
}
