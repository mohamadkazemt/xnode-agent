package limits

import (
	"context"
	"fmt"

	"xnode-agent/internal/model"
)

// Backend intentionally lives outside Xray config generation. Strict per-user
// speed/IP/connection enforcement requires a kernel backend or a small Xray patch.
type Backend interface {
	ApplyUser(ctx context.Context, nodeID, inboundID, userID string, l model.UserLimits) error
	RemoveUser(ctx context.Context, nodeID, inboundID, userID string) error
}

type ObserveOnly struct{}

func (ObserveOnly) ApplyUser(_ context.Context, _, _, _ string, l model.UserLimits) error {
	if l.UploadBPS > 0 || l.DownloadBPS > 0 || l.IPLimit > 0 || l.ConnectionLimit > 0 {
		return fmt.Errorf("strict limiter backend not configured")
	}
	return nil
}
func (ObserveOnly) RemoveUser(context.Context, string, string, string) error { return nil }
