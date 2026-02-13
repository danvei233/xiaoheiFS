package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type APIClient struct {
	baseURL string
	client  *http.Client
}

type RuntimeConfig struct {
	HeartbeatIntervalSec int `json:"heartbeat_interval_sec"`
	SnapshotIntervalSec  int `json:"snapshot_interval_sec"`
	LogChunkMaxBytes     int `json:"log_chunk_max_bytes"`
}

func New(baseURL string, insecure bool) *APIClient {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	return &APIClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		client: &http.Client{
			Timeout:   20 * time.Second,
			Transport: tr,
		},
	}
}

func (c *APIClient) Enroll(token, agentID, name, osType string) (probeID int64, probeSecret, accessToken string, cfg RuntimeConfig, err error) {
	payload := map[string]any{
		"enroll_token": token,
		"agent_id":     agentID,
		"name":         name,
		"os_type":      osType,
	}
	var resp struct {
		ProbeID     int64         `json:"probe_id"`
		ProbeSecret string        `json:"probe_secret"`
		AccessToken string        `json:"access_token"`
		Config      RuntimeConfig `json:"config"`
	}
	err = c.postJSON("/api/v1/probe/enroll", payload, &resp)
	return resp.ProbeID, resp.ProbeSecret, resp.AccessToken, resp.Config, err
}

func (c *APIClient) AuthToken(probeID int64, probeSecret string) (accessToken string, cfg RuntimeConfig, err error) {
	payload := map[string]any{
		"probe_id":     probeID,
		"probe_secret": probeSecret,
	}
	var resp struct {
		AccessToken string        `json:"access_token"`
		Config      RuntimeConfig `json:"config"`
	}
	err = c.postJSON("/api/v1/probe/auth/token", payload, &resp)
	return resp.AccessToken, resp.Config, err
}

func (c *APIClient) postJSON(path string, payload any, out any) error {
	b, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if out != nil {
		return json.Unmarshal(body, out)
	}
	return nil
}
