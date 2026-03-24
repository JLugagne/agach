package ws

import (
	"context"
	"encoding/json"
	"time"

	"github.com/JLugagne/agach-mcp/internal/daemon/app"
	"github.com/JLugagne/agach-mcp/pkg/daemonws"
	"github.com/sirupsen/logrus"
)

// ListDockerfilesResponse is the response payload for docker.list.
type ListDockerfilesResponse struct {
	Dockerfiles []DockerfileInfo `json:"dockerfiles"`
}

// DockerfileInfo describes a dockerfile and its build history.
type DockerfileInfo struct {
	Slug          string      `json:"slug"`
	LatestVersion string      `json:"latest_version"`
	VersionCount  int         `json:"version_count"`
	IsHealthy     bool        `json:"is_healthy"`
	Builds        []BuildInfo `json:"builds"`
}

// BuildInfo describes a single build.
type BuildInfo struct {
	Version   string    `json:"version"`
	IsLatest  bool      `json:"is_latest"`
	BuiltAt   time.Time `json:"built_at"`
	ImageHash string    `json:"image_hash"`
	Status    string    `json:"status"`
	SizeBytes int64     `json:"size_bytes"`
}

// GetLogsRequest is the payload for docker.logs.
type GetLogsRequest struct {
	BuildID string `json:"build_id"`
}

// GetLogsResponse is the response payload for docker.logs.
type GetLogsResponse struct {
	BuildID    string `json:"build_id"`
	Slug       string `json:"slug"`
	Version    string `json:"version"`
	Status     string `json:"status"`
	Log        string `json:"log"`
	InProgress bool   `json:"in_progress,omitempty"`
}

// DockerServicer defines the docker service operations used by handlers.
type DockerServicer interface {
	ListImages(ctx context.Context) ([]app.DockerfileWithBuilds, error)
	Rebuild(ctx context.Context, slug string, eventCh chan<- daemonws.BuildEvent) error
	GetBuildLogs(ctx context.Context, buildID string) (*app.BuildLogResult, error)
	PruneNonLatest(ctx context.Context, eventCh chan<- daemonws.PruneEvent) (app.PruneResult, error)
}

// Handlers holds WS message handlers and their dependencies.
type Handlers struct {
	docker DockerServicer
	hub    *Hub
	logger *logrus.Logger
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(docker DockerServicer, hub *Hub, logger *logrus.Logger) *Handlers {
	return &Handlers{docker: docker, hub: hub, logger: logger}
}

// RegisterAll registers all handlers with the hub.
func (h *Handlers) RegisterAll() {
	h.hub.RegisterHandler(daemonws.TypeDockerList, h.HandleListDockerfiles)
	h.hub.RegisterHandler(daemonws.TypeDockerRebuild, h.HandleRebuild)
	h.hub.RegisterHandler(daemonws.TypeDockerLogs, h.HandleGetLogs)
	h.hub.RegisterHandler(daemonws.TypeDockerPrune, h.HandlePrune)
}

// HandleListDockerfiles handles docker.list requests.
func (h *Handlers) HandleListDockerfiles(ctx context.Context, msg daemonws.Message) (daemonws.Message, error) {
	images, err := h.docker.ListImages(ctx)
	if err != nil {
		return daemonws.Message{}, err
	}

	resp := ListDockerfilesResponse{
		Dockerfiles: make([]DockerfileInfo, 0, len(images)),
	}

	for _, img := range images {
		info := DockerfileInfo{
			Slug:          img.Slug,
			LatestVersion: img.LatestTag,
			VersionCount:  len(img.Builds),
			IsHealthy:     img.IsHealthy,
			Builds:        make([]BuildInfo, 0, len(img.Builds)),
		}
		for _, b := range img.Builds {
			info.Builds = append(info.Builds, BuildInfo{
				Version:   b.Version,
				IsLatest:  b.Version == img.LatestTag,
				BuiltAt:   b.CreatedAt,
				ImageHash: b.ImageHash,
				Status:    string(b.Status),
				SizeBytes: b.ImageSize,
			})
		}
		resp.Dockerfiles = append(resp.Dockerfiles, info)
	}

	payload, err := marshalPayload(resp)
	if err != nil {
		return daemonws.Message{}, err
	}

	return daemonws.Message{
		Type:      daemonws.TypeDockerList,
		RequestID: msg.RequestID,
		Payload:   payload,
	}, nil
}

// RebuildRequest is the payload for docker.rebuild.
type RebuildRequest struct {
	Slug string `json:"slug"`
}

// HandleRebuild handles docker.rebuild requests.
func (h *Handlers) HandleRebuild(ctx context.Context, msg daemonws.Message) (daemonws.Message, error) {
	var req RebuildRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return daemonws.Message{
			Type:      daemonws.TypeError,
			RequestID: msg.RequestID,
			Error:     "invalid payload: " + err.Error(),
		}, nil
	}
	if req.Slug == "" {
		return daemonws.Message{
			Type:      daemonws.TypeError,
			RequestID: msg.RequestID,
			Error:     "slug is required",
		}, nil
	}

	// Start rebuild asynchronously
	go func() {
		eventCh := make(chan daemonws.BuildEvent, 10)
		go h.forwardBuildEvents(eventCh)
		if err := h.docker.Rebuild(ctx, req.Slug, eventCh); err != nil {
			h.logger.WithError(err).Error("rebuild failed")
		}
		close(eventCh)
	}()

	ack, _ := marshalPayload(map[string]string{
		"status": "started",
		"slug":   req.Slug,
	})
	return daemonws.Message{
		Type:      daemonws.TypeDockerRebuild,
		RequestID: msg.RequestID,
		Payload:   ack,
	}, nil
}

// HandleGetLogs handles docker.logs requests.
func (h *Handlers) HandleGetLogs(ctx context.Context, msg daemonws.Message) (daemonws.Message, error) {
	var req GetLogsRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return daemonws.Message{
			Type:      daemonws.TypeError,
			RequestID: msg.RequestID,
			Error:     "invalid payload: " + err.Error(),
		}, nil
	}
	if req.BuildID == "" {
		return daemonws.Message{
			Type:      daemonws.TypeError,
			RequestID: msg.RequestID,
			Error:     "build_id is required",
		}, nil
	}

	result, err := h.docker.GetBuildLogs(ctx, req.BuildID)
	if err != nil {
		return daemonws.Message{}, err
	}
	if result == nil {
		return daemonws.Message{
			Type:      daemonws.TypeError,
			RequestID: msg.RequestID,
			Error:     "build not found: " + req.BuildID,
		}, nil
	}

	resp := GetLogsResponse{
		BuildID:    result.BuildID,
		Slug:       result.Slug,
		Version:    result.Version,
		Status:     result.Status,
		Log:        result.Log,
		InProgress: result.Status == "building",
	}

	payload, err := marshalPayload(resp)
	if err != nil {
		return daemonws.Message{}, err
	}

	return daemonws.Message{
		Type:      daemonws.TypeDockerLogs,
		RequestID: msg.RequestID,
		Payload:   payload,
	}, nil
}

// HandlePrune handles docker.prune requests.
func (h *Handlers) HandlePrune(ctx context.Context, msg daemonws.Message) (daemonws.Message, error) {
	// Start prune asynchronously
	go func() {
		eventCh := make(chan daemonws.PruneEvent, 10)
		go h.forwardPruneEvents(eventCh)
		result, err := h.docker.PruneNonLatest(ctx, eventCh)
		close(eventCh)
		if err != nil {
			h.logger.WithError(err).Error("prune failed")
		} else {
			h.logger.WithField("removed", result.Removed).Info("prune completed")
		}
	}()

	ack, _ := marshalPayload(map[string]string{
		"status": "started",
	})
	return daemonws.Message{
		Type:      daemonws.TypeDockerPrune,
		RequestID: msg.RequestID,
		Payload:   ack,
	}, nil
}

// marshalPayload is a helper to marshal a response payload.
func marshalPayload(v any) (json.RawMessage, error) {
	return json.Marshal(v)
}

// forwardBuildEvents forwards build events from channel to hub as WS messages.
func (h *Handlers) forwardBuildEvents(eventCh <-chan daemonws.BuildEvent) {
	for event := range eventCh {
		payload, _ := marshalPayload(event)
		if h.hub != nil {
			h.hub.SendEvent(daemonws.Message{
				Type:    daemonws.TypeBuildEvent,
				Payload: payload,
			})
		}
	}
}

// forwardPruneEvents forwards prune events from channel to hub as WS messages.
func (h *Handlers) forwardPruneEvents(eventCh <-chan daemonws.PruneEvent) {
	for event := range eventCh {
		payload, _ := marshalPayload(event)
		if h.hub != nil {
			h.hub.SendEvent(daemonws.Message{
				Type:    daemonws.TypePruneEvent,
				Payload: payload,
			})
		}
	}
}
