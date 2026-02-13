package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2/google"

	"xiaoheiplay/internal/usecase"
)

const (
	fcmLegacyEndpoint = "https://fcm.googleapis.com/fcm/send"
	fcmOAuthScope     = "https://www.googleapis.com/auth/firebase.messaging"
)

type FCMSender struct {
	http *http.Client
}

func NewFCMSender() *FCMSender {
	return &FCMSender{
		http: &http.Client{Timeout: 8 * time.Second},
	}
}

func (s *FCMSender) Send(ctx context.Context, config usecase.PushConfig, tokens []string, payload usecase.PushPayload) error {
	if len(tokens) == 0 {
		return nil
	}

	projectID := strings.TrimSpace(config.ProjectID)
	serviceAccountJSON := strings.TrimSpace(config.ServiceAccountJSON)
	if projectID != "" && serviceAccountJSON != "" {
		for _, token := range tokens {
			if strings.TrimSpace(token) == "" {
				continue
			}
			if err := s.sendV1(ctx, projectID, serviceAccountJSON, token, payload); err != nil {
				return err
			}
		}
		return nil
	}

	serverKey := strings.TrimSpace(config.LegacyServerKey)
	if serverKey == "" {
		return nil
	}
	const batchSize = 500
	for i := 0; i < len(tokens); i += batchSize {
		end := i + batchSize
		if end > len(tokens) {
			end = len(tokens)
		}
		if err := s.sendLegacyBatch(ctx, serverKey, tokens[i:end], payload); err != nil {
			return err
		}
	}
	return nil
}

func (s *FCMSender) sendLegacyBatch(ctx context.Context, serverKey string, tokens []string, payload usecase.PushPayload) error {
	body := map[string]any{
		"registration_ids": tokens,
		"priority":         "high",
		"notification": map[string]any{
			"title": payload.Title,
			"body":  payload.Body,
		},
	}
	if len(payload.Data) > 0 {
		body["data"] = payload.Data
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fcmLegacyEndpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+serverKey)
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fcm send failed: status %d", resp.StatusCode)
	}
	return nil
}

func (s *FCMSender) sendV1(ctx context.Context, projectID, serviceAccountJSON, token string, payload usecase.PushPayload) error {
	creds, err := google.CredentialsFromJSON(ctx, []byte(serviceAccountJSON), fcmOAuthScope)
	if err != nil {
		return fmt.Errorf("fcm v1 credentials invalid: %w", err)
	}
	accessToken, err := creds.TokenSource.Token()
	if err != nil {
		return fmt.Errorf("fcm v1 oauth token failed: %w", err)
	}

	body := map[string]any{
		"message": map[string]any{
			"token": token,
			"notification": map[string]any{
				"title": payload.Title,
				"body":  payload.Body,
			},
		},
	}
	if len(payload.Data) > 0 {
		body["message"].(map[string]any)["data"] = payload.Data
	}
	raw, _ := json.Marshal(body)

	endpoint := "https://fcm.googleapis.com/v1/projects/" + url.PathEscape(projectID) + "/messages:send"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken.AccessToken)

	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respRaw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("fcm v1 send failed: status %d body=%s", resp.StatusCode, strings.TrimSpace(string(respRaw)))
	}
	return nil
}
