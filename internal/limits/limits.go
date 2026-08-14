package limits

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"xnode-agent/internal/model"
)

// Backend is an optional external mirror/extension hook. The bundled maintained
// Xray dispatcher overlay is the primary v1 strict limiter implementation.
type Backend interface {
	ApplyUser(ctx context.Context, nodeID, inboundID, userID string, l model.UserLimits) error
	RemoveUser(ctx context.Context, nodeID, inboundID, userID string) error
}

var ErrBackendUnavailable = errors.New("strict limiter backend not configured")

type ObserveOnly struct{}

func (ObserveOnly) ApplyUser(_ context.Context, _, _, _ string, l model.UserLimits) error {
	if l.UploadBPS > 0 || l.DownloadBPS > 0 || l.ConnectionLimit > 0 {
		return ErrBackendUnavailable
	}
	return nil
}
func (ObserveOnly) RemoveUser(context.Context, string, string, string) error { return nil }

type HTTPBackend struct {
	BaseURL string
	Client  *http.Client
}

func New(baseURL string) Backend {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return ObserveOnly{}
	}
	return &HTTPBackend{BaseURL: baseURL, Client: &http.Client{Timeout: 5 * time.Second}}
}

func (b *HTTPBackend) ApplyUser(ctx context.Context, nodeID, inboundID, userID string, l model.UserLimits) error {
	return b.do(ctx, http.MethodPut, nodeID, inboundID, userID, l)
}

func (b *HTTPBackend) RemoveUser(ctx context.Context, nodeID, inboundID, userID string) error {
	return b.do(ctx, http.MethodDelete, nodeID, inboundID, userID, nil)
}

func (b *HTTPBackend) do(ctx context.Context, method, nodeID, inboundID, userID string, body any) error {
	path := "/v1/limits/" + url.PathEscape(nodeID) + "/" + url.PathEscape(inboundID) + "/" + url.PathEscape(userID)
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, b.BaseURL+path, r)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := b.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("limiter backend status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}
