package http

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/safego"
)

type probeWSEnvelope struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type probeIDURI struct {
	ID int64 `uri:"id" binding:"required,gt=0"`
}

type probeLogSessionURI struct {
	ID  int64 `uri:"id" binding:"required,gt=0"`
	SID int64 `uri:"sid" binding:"required,gt=0"`
}

type adminProbesQuery struct {
	Keyword string `form:"keyword" binding:"omitempty,max=128"`
	Status  string `form:"status" binding:"omitempty,max=32"`
}

type probeRefreshQuery struct {
	Refresh string `form:"refresh" binding:"omitempty,oneof=0 1"`
}

type probeSLAQuery struct {
	Days *int `form:"days" binding:"omitempty,gte=1,lte=365"`
}

func (h *Handler) ProbeEnroll(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrProbeDisabled.Error()})
		return
	}
	var payload struct {
		EnrollToken string `json:"enroll_token" binding:"required,max=128"`
		AgentID     string `json:"agent_id" binding:"required,max=128"`
		Name        string `json:"name" binding:"omitempty,max=128"`
		OSType      string `json:"os_type" binding:"required,max=32"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	node, secret, err := h.probeSvc.Enroll(c.Request.Context(), payload.EnrollToken, payload.AgentID, payload.Name, payload.OSType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	accessToken, err := h.signProbeToken(node.ID, time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"probe_id":     node.ID,
		"probe_secret": secret,
		"access_token": accessToken,
		"expires_in":   3600,
		"config":       h.probeRuntimeConfig(c.Request.Context()),
	})
}

func (h *Handler) ProbeAuthToken(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrProbeDisabled.Error()})
		return
	}
	var payload struct {
		ProbeID     int64  `json:"probe_id" binding:"required,gt=0"`
		ProbeSecret string `json:"probe_secret" binding:"required,max=256"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	node, err := h.probeSvc.ValidateSecret(c.Request.Context(), payload.ProbeID, payload.ProbeSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidCredential.Error()})
		return
	}
	accessToken, err := h.signProbeToken(node.ID, time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"probe_id":     node.ID,
		"access_token": accessToken,
		"expires_in":   3600,
		"config":       h.probeRuntimeConfig(c.Request.Context()),
	})
}

const (
	// probeWSMaxMessageBytes is the hard cap on inbound WS frames (4 MiB).
	probeWSMaxMessageBytes = 4 << 20
	// maxProbeLogChunkBytes caps a single log chunk (1 MiB).
	maxProbeLogChunkBytes = 1 << 20
	// maxProbeSnapshotBytes caps a single snapshot payload.
	maxProbeSnapshotBytes = 4 << 20
)

func (h *Handler) ProbeWS(c *gin.Context) {
	if h.probeSvc == nil || h.probeHub == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrProbeDisabled.Error()})
		return
	}
	probeID, err := h.parseProbeIDFromBearer(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
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
	conn.SetReadLimit(probeWSMaxMessageBytes)

	var writeMu sync.Mutex
	send := func(b []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
		return conn.WriteMessage(websocket.TextMessage, b)
	}

	connID := h.probeHub.RegisterConn(probeID, send)
	_ = h.probeSvc.MarkOnline(c.Request.Context(), probeID, "ws_connected")
	defer func() {
		// Tear down exactly this connection and mark the probe offline only when
		// the hub reports (atomically, under its lock) that no connection remains.
		// A reconnect's newer connection registered between the teardown and the
		// DB write keeps the probe online and is never wrongly marked offline.
		h.teardownProbeConn(probeID, connID)
		_ = conn.Close()
	}()

	hello := map[string]any{
		"type": "hello_ack",
		"payload": map[string]any{
			"probe_id": probeID,
			"config":   h.probeRuntimeConfig(c.Request.Context()),
		},
	}
	_ = h.probeHub.SendJSON(probeID, hello)

	// Keep WS downstream traffic active so probe-side read deadline does not
	// timeout on quiet links/proxies.
	stopPing := make(chan struct{})
	defer close(stopPing)
	safego.Go("probe websocket ping", func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopPing:
				return
			case <-c.Request.Context().Done():
				return
			case <-ticker.C:
				safego.Run("probe websocket ping tick", func() {
					_ = h.probeHub.SendJSON(probeID, map[string]any{
						"type":       "ping",
						"request_id": "ping_" + probeRandomToken(8),
						"payload": map[string]any{
							"at": time.Now().Format(time.RFC3339),
						},
					})
				})
			}
		}
	})

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
			_ = h.probeSvc.MarkOnline(c.Request.Context(), probeID, "hello")
		case "heartbeat":
			var payload struct {
				At string `json:"at"`
			}
			_ = json.Unmarshal(msg.Payload, &payload)
			at := parseTimeOrNow(payload.At)
			_ = h.probeSvc.HandleHeartbeat(c.Request.Context(), probeID, at)
		case "snapshot":
			var payload struct {
				At       string          `json:"at"`
				OSType   string          `json:"os_type"`
				Snapshot json.RawMessage `json:"snapshot"`
			}
			_ = json.Unmarshal(msg.Payload, &payload)
			snapshotRaw := strings.TrimSpace(string(payload.Snapshot))
			if len(snapshotRaw) > maxProbeSnapshotBytes {
				log.Printf("probe snapshot dropped probe_id=%d bytes=%d", probeID, len(snapshotRaw))
				continue
			}
			if snapshotRaw == "" {
				snapshotRaw = "{}"
			}
			at := parseTimeOrNow(payload.At)
			if err := h.probeSvc.HandleSnapshot(c.Request.Context(), probeID, at, snapshotRaw, payload.OSType); err != nil {
				log.Printf("probe snapshot save failed probe_id=%d bytes=%d err=%v", probeID, len(snapshotRaw), err)
			}
		case "log_chunk":
			var payload struct {
				SessionID string `json:"session_id"`
				Chunk     string `json:"chunk"`
			}
			_ = json.Unmarshal(msg.Payload, &payload)
			sid := strings.TrimSpace(payload.SessionID)
			chunk := payload.Chunk
			if len(chunk) > maxProbeLogChunkBytes {
				log.Printf("probe log chunk dropped probe_id=%d session_id=%s bytes=%d", probeID, sid, len(chunk))
				continue
			}
			// Ownership check: a probe may only publish into its own session.
			if !h.probeLogSessionOwnedBy(c.Request.Context(), sid, probeID) {
				log.Printf("probe log chunk rejected probe_id=%d session_id=%s", probeID, sid)
				continue
			}
			h.probeHub.PublishLogChunk(probeID, sid, msg.RequestID, chunk)
		case "log_end":
			var payload struct {
				SessionID string `json:"session_id"`
			}
			_ = json.Unmarshal(msg.Payload, &payload)
			sid := strings.TrimSpace(payload.SessionID)
			if !h.probeLogSessionOwnedBy(c.Request.Context(), sid, probeID) {
				log.Printf("probe log end rejected probe_id=%d session_id=%s", probeID, sid)
				continue
			}
			h.probeHub.PublishLogEnd(probeID, sid, msg.RequestID)
			if sidInt, _ := strconv.ParseInt(sid, 10, 64); sidInt > 0 {
				_ = h.probeSvc.FinishLogSession(c.Request.Context(), sidInt, "done")
			}
		case "pong":
			_ = h.probeSvc.HandleHeartbeat(c.Request.Context(), probeID, time.Now())
		}
	}
}

func (h *Handler) AdminProbes(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrProbeDisabled.Error()})
		return
	}
	var query adminProbesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	limit, offset := paging(c)
	items, total, err := h.probeSvc.ListProbes(c.Request.Context(), appshared.ProbeNodeFilter{
		Keyword: query.Keyword,
		Status:  query.Status,
	}, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListProbeError.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrProbeDisabled.Error()})
		return
	}
	var payload struct {
		Name    string   `json:"name" binding:"required,max=128"`
		AgentID string   `json:"agent_id" binding:"required,max=128"`
		OSType  string   `json:"os_type" binding:"required,max=32"`
		Tags    []string `json:"tags"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	node, enrollToken, err := h.probeSvc.CreateProbe(c.Request.Context(), payload.Name,
		payload.AgentID,
		payload.OSType,
		mustJSON(payload.Tags),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c.Request.Context(), getUserID(c), "probe.create", "probe", strconv.FormatInt(node.ID, 10), map[string]any{"agent_id": node.AgentID})
	}
	c.JSON(http.StatusOK, gin.H{"probe": toProbeNodeDTO(node), "enroll_token": enrollToken})
}

func (h *Handler) AdminProbeDetail(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrProbeDisabled.Error()})
		return
	}
	var uri probeIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var query probeRefreshQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	node, err := h.probeSvc.GetProbe(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrProbeNotFound.Error()})
		return
	}
	if query.Refresh == "1" && h.probeHub != nil && h.probeHub.IsOnline(uri.ID) {
		reqID := "snap_" + probeRandomToken(10)
		_ = h.probeHub.SendJSON(uri.ID, map[string]any{
			"type":       "request_snapshot",
			"request_id": reqID,
			"payload":    map[string]any{},
		})
		prev := node.LastSnapshotAt
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			time.Sleep(200 * time.Millisecond)
			latest, getErr := h.probeSvc.GetProbe(c.Request.Context(), uri.ID)
			if getErr != nil {
				break
			}
			if latest.LastSnapshotAt != nil && (prev == nil || latest.LastSnapshotAt.After(*prev)) {
				node = latest
				break
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"probe": toProbeNodeDTO(node), "online": h.probeHub != nil && h.probeHub.IsOnline(uri.ID)})
}

func (h *Handler) AdminProbeUpdate(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrProbeDisabled.Error()})
		return
	}
	var uri probeIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	node, err := h.probeSvc.GetProbe(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrProbeNotFound.Error()})
		return
	}
	var payload struct {
		Name   *string   `json:"name" binding:"omitempty,max=128"`
		OSType *string   `json:"os_type" binding:"omitempty,max=32"`
		Tags   *[]string `json:"tags"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	if payload.Name != nil {
		node.Name = strings.TrimSpace(*payload.Name)
		if node.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
			return
		}
	}
	if payload.OSType != nil {
		node.OSType = strings.TrimSpace(*payload.OSType)
	}
	if payload.Tags != nil {
		node.TagsJSON = mustJSON(*payload.Tags)
	}
	if err := h.probeSvc.UpdateProbe(c.Request.Context(), node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c.Request.Context(), getUserID(c), "probe.update", "probe", strconv.FormatInt(node.ID, 10), map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminProbeDelete(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrProbeDisabled.Error()})
		return
	}
	var uri probeIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if _, err := h.probeSvc.GetProbe(c.Request.Context(), uri.ID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrProbeNotFound.Error()})
		return
	}
	if err := h.probeSvc.DeleteProbe(c.Request.Context(), uri.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.probeHub != nil {
		h.probeHub.RemoveProbe(uri.ID)
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c.Request.Context(), getUserID(c), "probe.delete", "probe", strconv.FormatInt(uri.ID, 10), map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminProbeResetEnrollToken(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrProbeDisabled.Error()})
		return
	}
	var uri probeIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if _, err := h.probeSvc.GetProbe(c.Request.Context(), uri.ID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrProbeNotFound.Error()})
		return
	}
	token, err := h.probeSvc.ResetEnrollToken(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enroll_token": token})
}

func (h *Handler) AdminProbeSLA(c *gin.Context) {
	if h.probeSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrProbeDisabled.Error()})
		return
	}
	var uri probeIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var query probeSLAQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	days := 7
	if query.Days != nil {
		days = *query.Days
	}
	sla, err := h.probeSvc.ComputeSLA(c.Request.Context(), uri.ID, days)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrProbeDisabled.Error()})
		return
	}
	var uri probeIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if !h.probeHub.IsOnline(uri.ID) {
		c.JSON(http.StatusConflict, gin.H{"error": domain.ErrProbeOffline.Error()})
		return
	}
	reqID := "pc_" + probeRandomToken(12)
	var payload map[string]any
	if err := bindJSONOptional(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	cmd := map[string]any{
		"type":       "port_check_request",
		"request_id": reqID,
		"payload":    payload,
	}
	if err := h.probeHub.SendJSON(uri.ID, cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "request_id": reqID})
}

func (h *Handler) AdminProbeLogSessionCreate(c *gin.Context) {
	if h.probeSvc == nil || h.probeHub == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrProbeDisabled.Error()})
		return
	}
	var uri probeIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if !h.probeHub.IsOnline(uri.ID) {
		c.JSON(http.StatusConflict, gin.H{"error": domain.ErrProbeOffline.Error()})
		return
	}
	var payload struct {
		Source  string `json:"source" binding:"omitempty,max=256"`
		Keyword string `json:"keyword" binding:"omitempty,max=1024"`
		Follow  bool   `json:"follow"`
		Lines   int    `json:"lines" binding:"omitempty,min=1,max=100000"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	source := h.resolveProbeLogSource(c.Request.Context(), payload.Source)
	session, err := h.probeSvc.CreateLogSession(c.Request.Context(), uri.ID, getUserID(c), source)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sid := strconv.FormatInt(session.ID, 10)
	ttl := time.Duration(h.probeIntSetting(c.Request.Context(), "probe_log_session_ttl_sec", 600, 1, 3600)) * time.Second
	h.probeHub.OpenLogSession(sid, uri.ID, ttl)
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
	if err := h.probeHub.SendJSON(uri.ID, cmd); err != nil {
		h.probeHub.CloseLogSession(sid)
		_ = h.probeSvc.FinishLogSession(c.Request.Context(), session.ID, "failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"session_id":   sid,
		"stream_path":  fmt.Sprintf("/admin/api/v1/probes/%d/log-sessions/%s/stream", uri.ID, sid),
		"log_session":  session,
		"request_id":   reqID,
		"probe_online": true,
	})
}

func (h *Handler) AdminProbeLogSessionStream(c *gin.Context) {
	if h.probeSvc == nil || h.probeHub == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrProbeDisabled.Error()})
		return
	}
	var uri probeLogSessionURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	sid := strconv.FormatInt(uri.SID, 10)
	session, err := h.probeSvc.GetLogSession(c.Request.Context(), uri.SID)
	if err != nil || session.ProbeID != uri.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrSessionNotFound.Error()})
		return
	}
	ch, cancel, err := h.probeHub.SubscribeLogSession(sid)
	if err != nil {
		c.JSON(http.StatusGone, gin.H{"error": domain.ErrSessionClosed.Error()})
		return
	}
	defer cancel()

	clearWriteDeadline(c.Writer)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrStreamUnsupported.Error()})
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
		"iss":      jwtIssuer,
		"aud":      jwtAudience,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(ttl).Unix(),
	})
	return token.SignedString(h.jwtSecret)
}

func (h *Handler) parseProbeIDFromBearer(auth string) (int64, error) {
	auth = strings.TrimSpace(auth)
	if !strings.HasPrefix(auth, "Bearer ") {
		return 0, domain.ErrMissingBearerToken
	}
	raw := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	if raw == "" {
		return 0, domain.ErrEmptyToken
	}
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(raw, claims,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, domain.ErrUnexpectedSigningMethod
			}
			return h.jwtSecret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(jwtIssuer),
		jwt.WithAudience(jwtAudience),
	)
	if err != nil || parsed == nil || !parsed.Valid {
		return 0, domain.ErrInvalidToken
	}
	if tt, _ := claims["type"].(string); tt != "probe_access" {
		return 0, domain.ErrInvalidTokenType
	}
	role, _ := claims["role"].(string)
	if role != "probe" {
		return 0, domain.ErrInvalidRole
	}
	probeID, ok := parseMapInt64(claims["probe_id"])
	if !ok || probeID <= 0 {
		return 0, domain.ErrInvalidProbeID
	}
	return probeID, nil
}

// parseTimeOrNow parses a client-supplied RFC3339 timestamp and clamps it to a
// bounded window: future timestamps beyond 5 minutes and past timestamps older
// than 24h both fall back to now.
func parseTimeOrNow(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Now()
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Now()
	}
	now := time.Now()
	if t.After(now.Add(5 * time.Minute)) {
		return now
	}
	if t.Before(now.Add(-24 * time.Hour)) {
		return now
	}
	return t
}

// teardownProbeConn removes the connection from the hub and marks the probe
// offline only when the hub reports (atomically with the removal) that no
// connection remains. Returns true when the probe is still online.
func (h *Handler) teardownProbeConn(probeID int64, connID uint64) bool {
	if h.probeHub == nil {
		return false
	}
	stillOnline := h.probeHub.UnregisterConn(probeID, connID)
	if !stillOnline && h.probeSvc != nil {
		_ = h.probeSvc.MarkOffline(context.Background(), probeID, "ws_disconnected")
	}
	return stillOnline
}

// probeLogSessionOwnedBy reports whether the given log session belongs to the
// given probe. The hub's in-memory session map is authoritative during active
// streaming (OpenLogSession is called for every session), so an active session
// is decided with zero DB round-trips — log_chunk frames do not become per-frame
// DB queries. Only sessions unknown to the hub (e.g. the hub entry expired) fall
// back to the DB so a late log_end can still finish a DB session.
func (h *Handler) probeLogSessionOwnedBy(ctx context.Context, sessionID string, probeID int64) bool {
	if h.probeHub != nil {
		if h.probeHub.HasLogSession(sessionID) {
			return h.probeHub.LogSessionOwnedBy(sessionID, probeID)
		}
	}
	sid, err := strconv.ParseInt(strings.TrimSpace(sessionID), 10, 64)
	if err != nil || sid <= 0 {
		return false
	}
	if h.probeSvc == nil {
		return false
	}
	session, err := h.probeSvc.GetLogSession(ctx, sid)
	if err != nil {
		return false
	}
	return session.ProbeID == probeID
}

func (h *Handler) probeRuntimeConfig(ctx context.Context) map[string]any {
	return map[string]any{
		"heartbeat_interval_sec": h.probeIntSetting(ctx, "probe_heartbeat_interval_sec", 20, 1, 3600),
		"snapshot_interval_sec":  h.probeIntSetting(ctx, "probe_snapshot_interval_sec", 60, 1, 3600),
		"log_chunk_max_bytes":    h.probeIntSetting(ctx, "probe_log_chunk_max_bytes", 16384, 1024, maxProbeLogChunkBytes),
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
	if h.settingsSvc == nil && h.adminSvc == nil {
		return fallback
	}
	item, err := h.getSettingByContext(ctx, "probe_log_file_source")
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

func (h *Handler) probeIntSetting(ctx context.Context, key string, def, min, max int) int {
	if h.settingsSvc == nil && h.adminSvc == nil {
		return clampInt(def, min, max)
	}
	item, err := h.getSettingByContext(ctx, key)
	if err != nil {
		return clampInt(def, min, max)
	}
	v, err := strconv.Atoi(strings.TrimSpace(item.ValueJSON))
	if err != nil {
		return clampInt(def, min, max)
	}
	return clampInt(v, min, max)
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func probeRandomToken(n int) string {
	if n <= 0 {
		return ""
	}
	letters := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	one := make([]byte, 1)
	maxByte := byte(256 - 256%len(letters))
	buf := make([]byte, n)
	for i := 0; i < n; i++ {
		for {
			_, _ = rand.Read(one)
			if one[0] < maxByte {
				break
			}
		}
		buf[i] = letters[int(one[0])%len(letters)]
	}
	return string(buf)
}
