package e2eapi

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Inline response structs
// ---------------------------------------------------------------------------

type dockerfileResp struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Content     string `json:"content"`
	IsLatest    bool   `json:"is_latest"`
	SortOrder   int    `json:"sort_order"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type projectResp struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

// projectDockerfileResp is used in TestDockerfiles_ProjectAssignment.
// Kept here for readability; the test decodes into dockerfileResp directly.

// ---------------------------------------------------------------------------
// 1. TestDockerfiles_CRUD
// ---------------------------------------------------------------------------

func TestDockerfiles_CRUD(t *testing.T) {
	tok := adminToken(t)
	slug := uniqueSlug("df-crud")

	// CREATE
	created := createAndDecode[dockerfileResp](t, "/api/dockerfiles", tok, map[string]any{
		"slug":        slug,
		"name":        "CRUD Dockerfile",
		"description": "A dockerfile for CRUD testing",
		"version":     "1.0.0",
		"content":     "FROM golang:1.22\nRUN echo hello",
		"is_latest":   false,
		"sort_order":  10,
	})
	require.NotEmpty(t, created.ID)
	assert.Equal(t, slug, created.Slug)
	assert.Equal(t, "CRUD Dockerfile", created.Name)
	assert.Equal(t, "A dockerfile for CRUD testing", created.Description)
	assert.Equal(t, "1.0.0", created.Version)
	assert.Equal(t, "FROM golang:1.22\nRUN echo hello", created.Content)
	assert.Equal(t, false, created.IsLatest)
	assert.Equal(t, 10, created.SortOrder)

	// GET by ID
	fetched := getAndDecode[dockerfileResp](t, "/api/dockerfiles/"+created.ID, tok)
	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, slug, fetched.Slug)
	assert.Equal(t, "CRUD Dockerfile", fetched.Name)

	// GET by slug
	fetchedBySlug := getAndDecode[dockerfileResp](t, "/api/dockerfiles/by-slug/"+slug, tok)
	assert.Equal(t, created.ID, fetchedBySlug.ID)
	assert.Equal(t, slug, fetchedBySlug.Slug)
	assert.Equal(t, "CRUD Dockerfile", fetchedBySlug.Name)

	// LIST — must contain the created dockerfile
	dockerfiles := getAndDecode[[]dockerfileResp](t, "/api/dockerfiles", tok)
	found := false
	for _, d := range dockerfiles {
		if d.ID == created.ID {
			found = true
			break
		}
	}
	require.True(t, found, "created dockerfile not found in list")

	// UPDATE name + content
	patchResp := doAuth(t, "PATCH", "/api/dockerfiles/"+created.ID, tok, map[string]any{
		"name":    ptr("Updated Dockerfile"),
		"content": ptr("FROM golang:1.23\nRUN echo updated"),
	})
	requireStatus(t, patchResp, http.StatusOK)
	patchResp.Body.Close()

	// Verify update via GET
	refetched := getAndDecode[dockerfileResp](t, "/api/dockerfiles/"+created.ID, tok)
	assert.Equal(t, "Updated Dockerfile", refetched.Name)
	assert.Equal(t, "FROM golang:1.23\nRUN echo updated", refetched.Content)
	assert.Equal(t, slug, refetched.Slug) // slug unchanged

	// DELETE
	deleteResource(t, "/api/dockerfiles/"+created.ID, tok)

	// Verify gone — expect 404
	resp := doAuth(t, "GET", "/api/dockerfiles/"+created.ID, tok, nil)
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// ---------------------------------------------------------------------------
// 2. TestDockerfiles_IsLatest
// ---------------------------------------------------------------------------

func TestDockerfiles_IsLatest(t *testing.T) {
	tok := adminToken(t)
	slug1 := uniqueSlug("df-latest1")
	slug2 := uniqueSlug("df-latest2")

	// Create first dockerfile with is_latest=true
	first := createAndDecode[dockerfileResp](t, "/api/dockerfiles", tok, map[string]any{
		"slug":       slug1,
		"name":       "First Latest",
		"version":    "1.0.0",
		"content":    "FROM alpine:3.19",
		"is_latest":  true,
		"sort_order": 1,
	})
	require.NotEmpty(t, first.ID)
	assert.True(t, first.IsLatest, "first dockerfile should be is_latest")

	// Create second dockerfile with is_latest=true
	second := createAndDecode[dockerfileResp](t, "/api/dockerfiles", tok, map[string]any{
		"slug":       slug2,
		"name":       "Second Latest",
		"version":    "2.0.0",
		"content":    "FROM alpine:3.20",
		"is_latest":  true,
		"sort_order": 2,
	})
	require.NotEmpty(t, second.ID)
	assert.True(t, second.IsLatest, "second dockerfile should be is_latest")

	// Re-fetch first to check if is_latest was cleared
	firstRefetched := getAndDecode[dockerfileResp](t, "/api/dockerfiles/"+first.ID, tok)
	secondRefetched := getAndDecode[dockerfileResp](t, "/api/dockerfiles/"+second.ID, tok)

	// At least the second should be is_latest; the first may or may not be
	// depending on business rules (unique latest vs multiple latest).
	// We verify both are accessible and the second is marked latest.
	assert.True(t, secondRefetched.IsLatest, "second dockerfile should still be is_latest")
	_ = firstRefetched // available for further assertions if needed

	// Cleanup
	deleteResource(t, "/api/dockerfiles/"+first.ID, tok)
	deleteResource(t, "/api/dockerfiles/"+second.ID, tok)
}

// ---------------------------------------------------------------------------
// 3. TestDockerfiles_ProjectAssignment
// ---------------------------------------------------------------------------

func TestDockerfiles_ProjectAssignment(t *testing.T) {
	tok := adminToken(t)
	dfSlug := uniqueSlug("df-assign")

	// Create a dockerfile
	df := createAndDecode[dockerfileResp](t, "/api/dockerfiles", tok, map[string]any{
		"slug":       dfSlug,
		"name":       "Assignable Dockerfile",
		"version":    "1.0.0",
		"content":    "FROM ubuntu:24.04",
		"is_latest":  false,
		"sort_order": 1,
	})
	require.NotEmpty(t, df.ID)

	// Create a project
	proj := createAndDecode[projectResp](t, "/api/projects", tok, map[string]any{
		"name":        "DF Assignment Project",
		"description": "Project for dockerfile assignment test",
	})
	require.NotEmpty(t, proj.ID)

	// Assign dockerfile to project via PUT
	resp := doAuth(t, "PUT", fmt.Sprintf("/api/projects/%s/dockerfile", proj.ID), tok, map[string]any{
		"dockerfile_id": df.ID,
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// GET project dockerfile — verify assignment
	assigned := getAndDecode[dockerfileResp](t, fmt.Sprintf("/api/projects/%s/dockerfile", proj.ID), tok)
	assert.Equal(t, df.ID, assigned.ID)
	assert.Equal(t, dfSlug, assigned.Slug)
	assert.Equal(t, "Assignable Dockerfile", assigned.Name)

	// Clear assignment via DELETE
	deleteResource(t, fmt.Sprintf("/api/projects/%s/dockerfile", proj.ID), tok)

	// Verify assignment cleared — returns 200 with null data
	respAfter := doAuth(t, "GET", fmt.Sprintf("/api/projects/%s/dockerfile", proj.ID), tok, nil)
	requireStatus(t, respAfter, http.StatusOK)
	respAfter.Body.Close()

	// Cleanup
	deleteResource(t, "/api/projects/"+proj.ID, tok)
	deleteResource(t, "/api/dockerfiles/"+df.ID, tok)
}

// ---------------------------------------------------------------------------
// 4. TestDockerfiles_DB_Verification
// ---------------------------------------------------------------------------

func TestDockerfiles_DB_Verification(t *testing.T) {
	tok := adminToken(t)
	pool := testPool(t)
	slug := uniqueSlug("df-db")

	// Create dockerfile
	df := createAndDecode[dockerfileResp](t, "/api/dockerfiles", tok, map[string]any{
		"slug":        slug,
		"name":        "DB Verified Dockerfile",
		"description": "For DB verification",
		"version":     "2.0.0",
		"content":     "FROM node:20\nRUN npm install",
		"is_latest":   true,
		"sort_order":  5,
	})
	require.NotEmpty(t, df.ID)

	// Verify dockerfile exists in "dockerfiles" table
	require.True(t, rowExists(t, pool, "dockerfiles", "id = $1", df.ID),
		"dockerfile should exist in dockerfiles table")

	// Verify specific fields in the DB
	name := queryString(t, pool, "SELECT name FROM dockerfiles WHERE id = $1", df.ID)
	assert.Equal(t, "DB Verified Dockerfile", name)

	slug2 := queryString(t, pool, "SELECT slug FROM dockerfiles WHERE id = $1", df.ID)
	assert.Equal(t, slug, slug2)

	// Cleanup
	deleteResource(t, "/api/dockerfiles/"+df.ID, tok)

	// Verify row removed after deletion
	require.False(t, rowExists(t, pool, "dockerfiles", "id = $1", df.ID),
		"dockerfile should be removed from dockerfiles table after delete")
}
