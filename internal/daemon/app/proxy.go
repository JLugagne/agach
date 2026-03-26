package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// SidecarProxy is a per-session Unix socket reverse proxy that forwards
// requests from the sidecar to the agach server, injecting authentication
// and project/feature context.
type SidecarProxy struct {
	socketPath   string
	apiKey       string
	projectID    string
	featureID    string
	serverURL    string
	getToken     func() string
	refreshToken func() error
	listener     net.Listener
	logger       *logrus.Logger
	mu           sync.Mutex
	connected    bool
	server       *http.Server
}

// NewSidecarProxy creates a new proxy instance. It does not start listening.
func NewSidecarProxy(
	socketPath, apiKey, projectID, featureID, serverURL string,
	getToken func() string,
	refreshToken func() error,
	logger *logrus.Logger,
) *SidecarProxy {
	return &SidecarProxy{
		socketPath:   socketPath,
		apiKey:       apiKey,
		projectID:    projectID,
		featureID:    featureID,
		serverURL:    serverURL,
		getToken:     getToken,
		refreshToken: refreshToken,
		logger:       logger,
	}
}

// Start creates the Unix socket and begins serving requests.
func (p *SidecarProxy) Start(ctx context.Context) error {
	// Remove stale socket file if it exists
	os.Remove(p.socketPath)

	ln, err := net.Listen("unix", p.socketPath)
	if err != nil {
		return fmt.Errorf("listen unix %s: %w", p.socketPath, err)
	}
	// Restrict socket permissions to owner only
	if err := os.Chmod(p.socketPath, 0600); err != nil {
		ln.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}
	p.listener = ln

	target, err := url.Parse(p.serverURL)
	if err != nil {
		ln.Close()
		return fmt.Errorf("parse server URL: %w", err)
	}

	proxy := &httputil.ReverseProxy{
		Director: p.director(target),
		ModifyResponse: func(resp *http.Response) error {
			// On 401, try refreshing token and signal retry
			if resp.StatusCode == http.StatusUnauthorized {
				if refreshErr := p.refreshToken(); refreshErr != nil {
					p.logger.WithError(refreshErr).Warn("token refresh failed on 401")
				}
			}
			return nil
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// API key authentication
		if r.Header.Get("X-Api-Key") != p.apiKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Single-connection enforcement
		p.mu.Lock()
		if !p.connected {
			p.connected = true
		}
		p.mu.Unlock()

		proxy.ServeHTTP(w, r)
	})

	p.server = &http.Server{Handler: mux}

	go func() {
		if err := p.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			p.logger.WithError(err).Error("sidecar proxy serve error")
		}
	}()

	p.logger.WithFields(logrus.Fields{
		"socket":     p.socketPath,
		"project_id": p.projectID,
		"feature_id": p.featureID,
	}).Info("Sidecar proxy started")

	return nil
}

// Stop shuts down the proxy and removes the socket file.
func (p *SidecarProxy) Stop() error {
	if p.server != nil {
		p.server.Close()
	}
	if p.listener != nil {
		p.listener.Close()
	}
	os.Remove(p.socketPath)
	p.logger.WithField("socket", p.socketPath).Info("Sidecar proxy stopped")
	return nil
}

// director returns the ReverseProxy Director function that rewrites requests.
func (p *SidecarProxy) director(target *url.URL) func(*http.Request) {
	return func(r *http.Request) {
		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host
		r.Host = target.Host

		// Replace "_" project ID placeholder with real project ID
		r.URL.Path = strings.Replace(r.URL.Path, "/api/projects/_/", "/api/projects/"+p.projectID+"/", 1)

		// Replace "_" feature ID placeholder with real feature ID
		if p.featureID != "" {
			r.URL.Path = strings.Replace(r.URL.Path, "/features/_/", "/features/"+p.featureID+"/", 1)
		}

		// Inject Bearer token
		r.Header.Set("Authorization", "Bearer "+p.getToken())

		// Remove the api key header before forwarding
		r.Header.Del("X-Api-Key")

		// Inject feature_id into task creation requests
		if p.featureID != "" && r.Method == http.MethodPost && isTaskCreationPath(r.URL.Path, p.projectID) {
			p.injectFeatureID(r)
		}
	}
}

// isTaskCreationPath checks if the path matches POST /api/projects/{id}/tasks (exact, no sub-resource)
func isTaskCreationPath(path, projectID string) bool {
	expected := "/api/projects/" + projectID + "/tasks"
	return path == expected
}

// injectFeatureID reads the request body JSON, adds "feature_id", and replaces the body.
func (p *SidecarProxy) injectFeatureID(r *http.Request) {
	if r.Body == nil {
		return
	}
	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		p.logger.WithError(err).Warn("failed to read request body for feature_id injection")
		r.Body = io.NopCloser(bytes.NewReader(body))
		return
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		r.Body = io.NopCloser(bytes.NewReader(body))
		return
	}

	// Only inject if not already set
	if _, ok := data["feature_id"]; !ok {
		data["feature_id"] = p.featureID
	}

	modified, err := json.Marshal(data)
	if err != nil {
		r.Body = io.NopCloser(bytes.NewReader(body))
		return
	}

	r.Body = io.NopCloser(bytes.NewReader(modified))
	r.ContentLength = int64(len(modified))
}

// generateAPIKey produces a cryptographically random 32-byte hex string.
func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate api key: %w", err)
	}
	return hex.EncodeToString(b), nil
}
