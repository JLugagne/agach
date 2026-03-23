package tui

import (
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// projectSelectedMsg is sent when the user picks a project on the welcome screen
type projectSelectedMsg struct {
	project pkgkanban.ProjectResponse
}

// rolesLoadedMsg carries the per-project roles
type rolesLoadedMsg struct {
	roles []pkgkanban.RoleResponse
	err   error
}

// configInitDoneMsg carries both init results
type configInitDoneMsg struct {
	roles    []pkgkanban.RoleResponse
	rolesErr error
	subs     []pkgkanban.ProjectResponse
	subsErr  error
}

// backToWelcomeMsg signals the user wants to go back to the welcome screen
type backToWelcomeMsg struct{}
