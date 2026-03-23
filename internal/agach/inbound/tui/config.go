package tui

import (
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

// projectSelectedMsg is sent when the user picks a project on the welcome screen
type projectSelectedMsg struct {
	project pkgserver.ProjectResponse
}

// rolesLoadedMsg carries the per-project roles
type rolesLoadedMsg struct {
	roles []pkgserver.RoleResponse
	err   error
}

// configInitDoneMsg carries both init results
type configInitDoneMsg struct {
	roles    []pkgserver.RoleResponse
	rolesErr error
	subs     []pkgserver.ProjectResponse
	subsErr  error
}

// backToWelcomeMsg signals the user wants to go back to the welcome screen
type backToWelcomeMsg struct{}
