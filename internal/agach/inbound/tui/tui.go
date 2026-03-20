package tui

import (
	"context"
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"

	appagach "github.com/JLugagne/agach-mcp/internal/agach/app"
	"github.com/JLugagne/agach-mcp/internal/agach/inbound/tui/tcellapp"
	"github.com/JLugagne/agach-mcp/pkg/kanban/client"
)

type screen int

const (
	screenWelcome screen = iota
	screenMonitor
	screenDiagnostic
)

// tuiApp holds shared application state across all screens
type tuiApp struct {
	kanban     *client.Client
	agach      *appagach.App
	serverURL  string
	workDir    string // current working directory
	tcellApp   *tcellapp.App
	runUpdates chan appagach.WorkerUpdate
	runCtx     context.Context
	cancelRun  context.CancelFunc
}

// RootScreen is the top-level tcellapp screen
type RootScreen struct {
	app        *tuiApp
	current    screen
	welcome    tcellapp.Screen
	monitor    tcellapp.Screen
	diagnostic tcellapp.Screen
}

func (r *RootScreen) Init() tcellapp.Cmd {
	return r.welcome.Init()
}

func (r *RootScreen) HandleMsg(msg tcellapp.Msg) (tcellapp.Screen, tcellapp.Cmd) {
	// Global quit
	if key, ok := msg.(tcellapp.KeyMsg); ok {
		ks := tcellapp.KeyString(key)
		if ks == "q" && r.current == screenWelcome {
			return r, func() tcellapp.Msg { return tcellapp.QuitMsg{} }
		}
		if ks == "ctrl+c" {
			return r, func() tcellapp.Msg { return tcellapp.QuitMsg{} }
		}
	}

	switch r.current {
	case screenWelcome:
		return r.handleWelcome(msg)
	case screenMonitor:
		return r.handleMonitor(msg)
	case screenDiagnostic:
		return r.handleDiagnostic(msg)
	}
	return r, nil
}

func (r *RootScreen) handleWelcome(msg tcellapp.Msg) (tcellapp.Screen, tcellapp.Cmd) {
	if ps, ok := msg.(projectSelectedMsg); ok {
		monitor := newMonitorModel(r.app, ps.project)
		r.monitor = monitor
		r.current = screenMonitor
		setTerminalTitle(ps.project.Name + " - agach")
		return r, r.monitor.Init()
	}

	if _, ok := msg.(launchDiagnosticMsg); ok {
		r.diagnostic = newDiagnosticModel(r.app)
		r.current = screenDiagnostic
		return r, r.diagnostic.Init()
	}

	newScreen, cmd := r.welcome.HandleMsg(msg)
	r.welcome = newScreen
	return r, cmd
}

func (r *RootScreen) handleMonitor(msg tcellapp.Msg) (tcellapp.Screen, tcellapp.Cmd) {
	if _, ok := msg.(backToWelcomeMsg); ok {
		r.current = screenWelcome
		setTerminalTitle("agach")
		return r, r.welcome.Init()
	}

	newScreen, cmd := r.monitor.HandleMsg(msg)
	r.monitor = newScreen
	return r, cmd
}

func (r *RootScreen) handleDiagnostic(msg tcellapp.Msg) (tcellapp.Screen, tcellapp.Cmd) {
	if _, ok := msg.(backToWelcomeMsg); ok {
		r.current = screenWelcome
		return r, r.welcome.Init()
	}

	newScreen, cmd := r.diagnostic.HandleMsg(msg)
	r.diagnostic = newScreen
	return r, cmd
}

func (r *RootScreen) Draw(s tcell.Screen, w, h int) {
	switch r.current {
	case screenWelcome:
		r.welcome.Draw(s, w, h)
	case screenMonitor:
		r.monitor.Draw(s, w, h)
	case screenDiagnostic:
		r.diagnostic.Draw(s, w, h)
	}
}

func setTerminalTitle(title string) {
	fmt.Fprintf(os.Stdout, "\033]0;%s\007", title)
}

// Run starts the TUI
func Run(serverURL string) error {
	workDir, _ := os.Getwd()
	app := &tuiApp{
		kanban:    client.New(serverURL),
		agach:     appagach.New(serverURL),
		serverURL: serverURL,
		workDir:   workDir,
	}

	root := &RootScreen{
		app:     app,
		current: screenWelcome,
		welcome: newWelcomeModel(app),
	}

	a, err := tcellapp.New(root)
	if err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	app.tcellApp = a
	return a.Run()
}

// EntryPoint is the main entry point for the agach CLI
func EntryPoint() {
	serverURL := os.Getenv("AGACH_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8322"
	}

	if err := Run(serverURL); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

