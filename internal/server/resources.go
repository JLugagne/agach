package server

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// ResourceEntry describes a single embedded resource with its checksum.
type ResourceEntry struct {
	Name   string `json:"name"`
	SHA512 string `json:"sha512"`
	Size   int64  `json:"size"`
}

// ResourceManifest holds the computed checksums for all embedded resources.
type ResourceManifest struct {
	entries []ResourceEntry
	fsys    fs.FS
	logger  *logrus.Logger
}

// ComputeManifest walks the embedded FS and computes SHA-512 for each file.
// It skips directories and the embed.go file itself.
func ComputeManifest(fsys fs.FS, logger *logrus.Logger) *ResourceManifest {
	m := &ResourceManifest{fsys: fsys, logger: logger}

	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		// Skip Go source files and gitkeep
		if path == "embed.go" || path == ".gitkeep" {
			return nil
		}

		f, err := fsys.Open(path)
		if err != nil {
			logger.WithError(err).WithField("path", path).Warn("skip resource: open failed")
			return nil
		}
		defer f.Close()

		h := sha512.New()
		size, err := io.Copy(h, f)
		if err != nil {
			logger.WithError(err).WithField("path", path).Warn("skip resource: hash failed")
			return nil
		}

		m.entries = append(m.entries, ResourceEntry{
			Name:   path,
			SHA512: hex.EncodeToString(h.Sum(nil)),
			Size:   size,
		})

		logger.WithFields(logrus.Fields{
			"resource": path,
			"size":     size,
		}).Info("Embedded resource indexed")

		return nil
	})

	return m
}

// Entries returns the manifest entries.
func (m *ResourceManifest) Entries() []ResourceEntry {
	return m.entries
}

// RegisterRoutes registers GET /api/resources and GET /api/resources/{name}.
func (m *ResourceManifest) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/resources", m.handleList).Methods("GET")
	router.HandleFunc("/api/resources/{name}", m.handleDownload).Methods("GET")
}

func (m *ResourceManifest) handleList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "success",
		"data":   m.entries,
	})
}

func (m *ResourceManifest) handleDownload(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	// Validate the name exists in the manifest
	found := false
	for _, e := range m.entries {
		if e.Name == name {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, `{"status":"fail","error":{"code":"NOT_FOUND","message":"resource not found"}}`, http.StatusNotFound)
		return
	}

	f, err := m.fsys.Open(name)
	if err != nil {
		http.Error(w, `{"status":"fail","error":{"code":"NOT_FOUND","message":"resource not found"}}`, http.StatusNotFound)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+name)
	io.Copy(w, f)
}
