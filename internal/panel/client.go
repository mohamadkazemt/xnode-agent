package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"xnode-agent/internal/model"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), token: token, http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("panel %s %s: status=%d body=%s", method, path, resp.StatusCode, string(b))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) DesiredState(ctx context.Context, nodeID string) (model.DesiredState, error) {
	var s model.DesiredState
	err := c.do(ctx, http.MethodGet, "/api/v1/nodes/"+nodeID+"/desired-state", nil, &s)
	return s, err
}

func (c *Client) Heartbeat(ctx context.Context, hb model.Heartbeat) error {
	return c.do(ctx, http.MethodPost, "/api/v1/nodes/"+hb.NodeID+"/heartbeat", hb, nil)
}

func (c *Client) Traffic(ctx context.Context, nodeID string, records []model.TrafficRecord) error {
	payload := map[string]any{"node_id": nodeID, "records": records}
	return c.do(ctx, http.MethodPost, "/api/v1/nodes/"+nodeID+"/traffic", payload, nil)
}
