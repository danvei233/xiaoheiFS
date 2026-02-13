package http

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"

	"xiaoheiplay/internal/usecase"
)

type probeWSEnvelope struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

func (h *Handler) ProbeEnroll(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "probe disabled"})
		return
	}
	var payload struct {
		EnrollToken string `json:"enroll_token"`
		AgentID     string `json:"agent_id"`
		Name        string `json:"name"`
		OSType      string `json:"os_type"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	node, secret, err := h.probeSvc.Enroll(c, payload.EnrollToken, payload.AgentID, payload.Name, payload.OSType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	accessToken, err := h.signProbeToken(node.ID, time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"probe_id":     node.ID,
		"probe_secret": secret,
		"access_token": accessToken,
		"expires_in":   3600,
		"config":       h.probeRuntimeConfig(c),
	})
}

func (h *Handler) ProbeAuthToken(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "probe disabled"})
		return
	}
	var payload struct {
		ProbeID     int64  `json:"probe_id"`
		ProbeSecret string `json:"probe_secret"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	node, err := h.probeSvc.ValidateSecret(c, payload.ProbeID, payload.ProbeSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credential"})
		return
	}
	accessToken, err := h.signProbeToken(node.ID, time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"probe_id":     node.ID,
		"access_token": accessToken,
		"expires_in":   3600,
		"config":       h.probeRuntimeConfig(c),
	})
}

func (h *Handler) ProbeWS(c *gin.Context) {
	if h.probeSvc == nil || h.probeHub == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "probe disabled"})
		return
	}
	probeID, err := h.parseProbeIDFromBearer(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	var writeMu sync.Mutex
	send := func(b []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
		return conn.WriteMessage(websocket.TextMessage, b)
	}

	h.probeHub.RegisterConn(probeID, send)
	_ = h.probeSvc.MarkOnline(c, probeID, "ws_connected")
	defer func() {
		h.probeHub.UnregisterConn(probeID)
		_ = h.probeSvc.MarkOffline(context.Background(), probeID, "ws_disconnected")
		_ = conn.Close()
	}()

	hello := map[string]any{
		"type": "hello_ack",
		"payload": map[string]any{
			"probe_id": probeID,
			"config":   h.probeRuntimeConfig(c),
		},
	}
	_ = h.probeHub.SendJSON(probeID, hello)

	// Keep WS downstream traffic active so probe-side read deadline does not
	// timeout on quiet links/proxies.
	stopPing := make(chan struct{})
	defer close(stopPing)
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopPing:
				return
			case <-c.Request.Context().Done():
				return
			case <-ticker.C:
				_ = h.probeHub.SendJSON(probeID, map[string]any{
					"type":       "ping",
					"request_id": "ping_" + probeRandomToken(8),
					"payload": map[string]any{
						"at": time.Now().Format(time.RFC3339),
					},
				})
			}
		}
	}()

	for {
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg probeWSEnvelope
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		switch strings.TrimSpace(msg.Type) {
		case "hello":
			_ = h.probeSvc.MarkOnline(c, probeID, "hello")
		case "heartbeat":
			var payload struct {
				At string `json:"at"`
			}
			_ = json.Unmarshal(msg.Payload, &payload)
			at := parseTimeOrNow(payload.At)
			_ = h.probeSvc.HandleHeartbeat(c, probeID, at)
		case "snapshot":
			var payload struct {
				At       string          `json:"at"`
				OSType   string          `json:"os_type"`
				Snapshot json.RawMessage `json:"snapshot"`
			}
			_ = json.Unmarshal(msg.Payload, &payload)
			snapshotRaw := strings.TrimSpace(string(payload.Snapshot))
			if snapshotRaw == "" {
				snapshotRaw = "{}"
			}
			at := parseTimeOrNow(payload.At)
			if err := h.probeSvc.HandleSnapshot(c, probeID, at, snapshotRaw, payload.OSType); err != nil {
				log.Printf("probe snapshot save failed probe_id=%d bytes=%d err=%v", probeID, len(snapshotRaw), err)
			}
		case "log_chunk":
			var payload struct {
				SessionID string `json:"session_id"`
				Chunk     string `json:"chunk"`
			}
			_ = json.Unmarshal(msg.Payload, &payload)
			h.probeHub.PublishLogChunk(strings.TrimSpace(payload.SessionID), msg.RequestID, payload.Chunk)
		case "log_end":
			var payload struct {
				SessionID string `json:"session_id"`
			}
			_ = json.Unmarshal(msg.Payload, &payload)
			h.probeHub.PublishLogEnd(strings.TrimSpace(payload.SessionID), msg.RequestID)
			if sid, _ := strconv.ParseInt(strings.TrimSpace(payload.SessionID), 10, 64); sid > 0 {
				_ = h.probeSvc.FinishLogSession(c, sid, "done")
			}
		case "pong":
			_ = h.probeSvc.HandleHeartbeat(c, probeID, time.Now())
		}
	}
}

func (h *Handler) AdminProbes(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "probe disabled"})
		return
	}
	limit, offset := paging(c)
	items, total, err := h.probeSvc.ListProbes(c, usecase.ProbeNodeFilter{
		Keyword: c.Query("keyword"),
		Status:  c.Query("status"),
	}, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list probe error"})
		return
	}
	out := make([]ProbeNodeDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toProbeNodeDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total})
}

func (h *Handler) AdminProbeCreate(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "probe disabled"})
		return
	}
	var payload struct {
		Name    string   `json:"name"`
		AgentID string   `json:"agent_id"`
		OSType  string   `json:"os_type"`
		Tags    []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	node, enrollToken, err := h.probeSvc.CreateProbe(
		c,
		payload.Name,
		payload.AgentID,
		payload.OSType,
		mustJSON(payload.Tags),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "probe.create", "probe", strconv.FormatInt(node.ID, 10), map[string]any{"agent_id": node.AgentID})
	}
	c.JSON(http.StatusOK, gin.H{"probe": toProbeNodeDTO(node), "enroll_token": enrollToken})
}

func (h *Handler) AdminProbeDetail(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "probe disabled"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	node, err := h.probeSvc.GetProbe(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "probe not found"})
		return
	}
	if c.Query("refresh") == "1" && h.probeHub != nil && h.probeHub.IsOnline(id) {
		reqID := "snap_" + probeRandomToken(10)
		_ = h.probeHub.SendJSON(id, map[string]any{
			"type":       "request_snapshot",
			"request_id": reqID,
			"payload":    map[string]any{},
		})
		prev := node.LastSnapshotAt
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			time.Sleep(200 * time.Millisecond)
			latest, getErr := h.probeSvc.GetProbe(c, id)
			if getErr != nil {
				break
			}
			if latest.LastSnapshotAt != nil && (prev == nil || latest.LastSnapshotAt.After(*prev)) {
				node = latest
				break
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"probe": toProbeNodeDTO(node), "online": h.probeHub != nil && h.probeHub.IsOnline(id)})
}

func (h *Handler) AdminProbeUpdate(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "probe disabled"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	node, err := h.probeSvc.GetProbe(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "probe not found"})
		return
	}
	var payload struct {
		Name   *string   `json:"name"`
		OSType *string   `json:"os_type"`
		Tags   *[]string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.Name != nil {
		node.Name = strings.TrimSpace(*payload.Name)
	}
	if payload.OSType != nil {
		node.OSType = strings.TrimSpace(*payload.OSType)
	}
	if payload.Tags != nil {
		node.TagsJSON = mustJSON(*payload.Tags)
	}
	if err := h.probeSvc.UpdateProbe(c, node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "probe.update", "probe", strconv.FormatInt(node.ID, 10), map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminProbeDelete(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "probe disabled"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if _, err := h.probeSvc.GetProbe(c, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "probe not found"})
		return
	}
	if err := h.probeSvc.DeleteProbe(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.probeHub != nil {
		h.probeHub.UnregisterConn(id)
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "probe.delete", "probe", strconv.FormatInt(id, 10), map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminProbeResetEnrollToken(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "probe disabled"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	token, err := h.probeSvc.ResetEnrollToken(c, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enroll_token": token})
}

func (h *Handler) AdminProbeSLA(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "probe disabled"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	days, _ := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("days", "7")))
	sla, err := h.probeSvc.ComputeSLA(c, id, days)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	evs := make([]ProbeStatusEventDTO, 0, len(sla.Events))
	for _, ev := range sla.Events {
		evs = append(evs, toProbeStatusEventDTO(ev))
	}
	c.JSON(http.StatusOK, gin.H{
		"sla": ProbeSLADTO{
			WindowFrom:    sla.WindowFrom,
			WindowTo:      sla.WindowTo,
			TotalSeconds:  sla.TotalSeconds,
			OnlineSeconds: sla.OnlineSeconds,
			UptimePercent: sla.UptimePercent,
			Events:        evs,
		},
	})
}

func (h *Handler) AdminProbePortCheck(c *gin.Context) {
	if h.probeSvc == nil || h.probeHub == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "probe disabled"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if !h.probeHub.IsOnline(id) {
		c.JSON(http.StatusConflict, gin.H{"error": "probe offline"})
		return
	}
	reqID := "pc_" + probeRandomToken(12)
	var payload map[string]any
	_ = c.ShouldBindJSON(&payload)
	cmd := map[string]any{
		"type":       "port_check_request",
		"request_id": reqID,
		"payload":    payload,
	}
	if err := h.probeHub.SendJSON(id, cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "request_id": reqID})
}

func (h *Handler) AdminProbeLogSessionCreate(c *gin.Context) {
	if h.probeSvc == nil || h.probeHub == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "probe disabled"})
		return
	}
	probeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if !h.probeHub.IsOnline(probeID) {
		c.JSON(http.StatusConflict, gin.H{"error": "probe offline"})
		return
	}
	var payload struct {
		Source  string `json:"source"`
		Keyword string `json:"keyword"`
		Follow  bool   `json:"follow"`
		Lines   int    `json:"lines"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	source := h.resolveProbeLogSource(c, payload.Source)
	session, err := h.probeSvc.CreateLogSession(c, probeID, getUserID(c), source)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sid := strconv.FormatInt(session.ID, 10)
	ttl := time.Duration(h.probeIntSetting(c, "probe_log_session_ttl_sec", 600)) * time.Second
	h.probeHub.OpenLogSession(sid, probeID, ttl)
	reqID := "log_" + probeRandomToken(12)
	cmd := map[string]any{
		"type":       "request_log",
		"request_id": reqID,
		"payload": map[string]any{
			"session_id": sid,
			"source":     source,
			"keyword":    payload.Keyword,
			"follow":     payload.Follow,
			"lines":      payload.Lines,
		},
	}
	if err := h.probeHub.SendJSON(probeID, cmd); err != nil {
		h.probeHub.CloseLogSession(sid)
		_ = h.probeSvc.FinishLogSession(c, session.ID, "failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"session_id":   sid,
		"stream_path":  fmt.Sprintf("/admin/api/v1/probes/%d/log-sessions/%s/stream", probeID, sid),
		"log_session":  session,
		"request_id":   reqID,
		"probe_online": true,
	})
}

func (h *Handler) AdminProbeLogSessionStream(c *gin.Context) {
	if h.probeSvc == nil || h.probeHub == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "probe disabled"})
		return
	}
	probeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	sid := strings.TrimSpace(c.Param("sid"))
	sessionID, _ := strconv.ParseInt(sid, 10, 64)
	session, err := h.probeSvc.GetLogSession(c, sessionID)
	if err != nil || session.ProbeID != probeID {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	ch, cancel, err := h.probeHub.SubscribeLogSession(sid)
	if err != nil {
		c.JSON(http.StatusGone, gin.H{"error": "session closed"})
		return
	}
	defer cancel()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stream unsupported"})
		return
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			dto := ProbeLogChunkDTO{
				Type:      msg.Type,
				RequestID: msg.RequestID,
				Data:      msg.Data,
				At:        msg.At,
			}
			body, _ := json.Marshal(dto)
			fmt.Fprintf(c.Writer, "event: %s\n", msg.Type)
			fmt.Fprintf(c.Writer, "data: %s\n\n", string(body))
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(c.Writer, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func (h *Handler) signProbeToken(probeID int64, ttl time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"probe_id": probeID,
		"role":     "probe",
		"type":     "probe_access",
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(ttl).Unix(),
	})
	return token.SignedString(h.jwtSecret)
}

func (h *Handler) parseProbeIDFromBearer(auth string) (int64, error) {
	auth = strings.TrimSpace(auth)
	if !strings.HasPrefix(auth, "Bearer ") {
		return 0, errors.New("missing bearer token")
	}
	raw := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	if raw == "" {
		return 0, errors.New("empty token")
	}
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return h.jwtSecret, nil
	})
	if err != nil || parsed == nil || !parsed.Valid {
		return 0, errors.New("invalid token")
	}
	role, _ := claims["role"].(string)
	if role != "probe" {
		return 0, errors.New("invalid role")
	}
	probeID, ok := parseMapInt64(claims["probe_id"])
	if !ok || probeID <= 0 {
		return 0, errors.New("invalid probe id")
	}
	return probeID, nil
}

func parseTimeOrNow(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Now()
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Now()
	}
	return t
}

func (h *Handler) probeRuntimeConfig(ctx context.Context) map[string]any {
	return map[string]any{
		"heartbeat_interval_sec": h.probeIntSetting(ctx, "probe_heartbeat_interval_sec", 20),
		"snapshot_interval_sec":  h.probeIntSetting(ctx, "probe_snapshot_interval_sec", 60),
		"log_chunk_max_bytes":    h.probeIntSetting(ctx, "probe_log_chunk_max_bytes", 16384),
	}
}

func (h *Handler) resolveProbeLogSource(ctx context.Context, source string) string {
	source = strings.TrimSpace(source)
	lower := strings.ToLower(source)
	if source == "" || strings.HasPrefix(lower, "file:") {
		return h.probeFileLogSource(ctx)
	}
	return source
}

func (h *Handler) probeFileLogSource(ctx context.Context) string {
	const fallback = "file:logs"
	if h.settings == nil {
		return fallback
	}
	item, err := h.settings.GetSetting(ctx, "probe_log_file_source")
	if err != nil {
		return fallback
	}
	value := strings.TrimSpace(item.ValueJSON)
	if value == "" {
		return fallback
	}
	if strings.HasPrefix(strings.ToLower(value), "file:") {
		return value
	}
	return "file:" + value
}

func (h *Handler) probeIntSetting(ctx context.Context, key string, def int) int {
	if h.settings == nil {
		return def
	}
	item, err := h.settings.GetSetting(ctx, key)
	if err != nil {
		return def
	}
	v, err := strconv.Atoi(strings.TrimSpace(item.ValueJSON))
	if err != nil {
		return def
	}
	return v
}

func probeRandomToken(n int) string {
	letters := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	for i := range buf {
		buf[i] = letters[int(buf[i])%len(letters)]
	}
	return string(buf)
}
