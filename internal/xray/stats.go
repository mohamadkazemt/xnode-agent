package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"

	"xnode-agent/internal/model"
)

type statsEnvelope struct {
	Stat []struct {
		Name  string `json:"name"`
		Value any    `json:"value"`
	} `json:"stat"`
}

func (m *Manager) QueryStats(ctx context.Context, reset bool) ([]model.TrafficRecord, error) {
	args := []string{"api", "statsquery", "--server=" + m.API}
	if reset {
		args = append(args, "-reset=true")
	}
	out, err := exec.CommandContext(ctx, m.Binary, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("statsquery: %w: %s", err, string(out))
	}
	return ParseStats(out)
}

func ParseStats(b []byte) ([]model.TrafficRecord, error) {
	var env statsEnvelope
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, err
	}
	res := make([]model.TrafficRecord, 0, len(env.Stat))
	for _, s := range env.Stat {
		v, err := toInt64(s.Value)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", s.Name, err)
		}
		r := model.TrafficRecord{Name: s.Name, Value: v}
		if u, i, d, ok := ParseAccountingName(s.Name); ok {
			r.UserID, r.InboundID, r.Direction = u, i, d
		}
		res = append(res, r)
	}
	return res, nil
}

func toInt64(v any) (int64, error) {
	switch x := v.(type) {
	case string:
		return strconv.ParseInt(x, 10, 64)
	case float64:
		return int64(x), nil
	default:
		return 0, fmt.Errorf("unsupported value type %T", v)
	}
}
