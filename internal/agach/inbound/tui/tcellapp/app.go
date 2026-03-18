package tcellapp

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// Msg is any value dispatched to the active screen.
type Msg = any

// Cmd is a function the screen schedules; its return value is dispatched as a Msg.
type Cmd func() Msg

// Screen is implemented by each TUI screen (welcome, config, monitor).
type Screen interface {
	// Init returns an optional Cmd to run on entry.
	Init() Cmd
	// HandleMsg processes a dispatched message and returns the updated screen
	// plus an optional follow-up Cmd.
	HandleMsg(Msg) (Screen, Cmd)
	// Draw renders the screen onto the tcell.Screen.
	Draw(s tcell.Screen, width, height int)
}

// Built-in message types.

// KeyMsg represents a key press event.
type KeyMsg struct {
	Key  tcell.Key
	Rune rune
	Mod  tcell.ModMask
}

// ResizeMsg represents a terminal resize event.
type ResizeMsg struct {
	Width, Height int
}

// QuitMsg signals the app to exit.
type QuitMsg struct{}

// NavigateMsg signals screen replacement.
type NavigateMsg struct {
	Next Screen
}

// SuspendMsg asks the main loop to suspend the terminal and run a function.
type SuspendMsg struct {
	Fn func()
}

// App runs the tcell event loop.
type App struct {
	screen  tcell.Screen
	current Screen
	msgs    chan Msg
}

// New creates a new App with the given initial screen.
func New(initial Screen) (*App, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("tcellapp: new screen: %w", err)
	}
	if err := s.Init(); err != nil {
		return nil, fmt.Errorf("tcellapp: init screen: %w", err)
	}
	s.EnableMouse()
	s.Clear()

	return &App{
		screen:  s,
		current: initial,
		msgs:    make(chan Msg, 64),
	}, nil
}

// Dispatch sends a message to the app's message channel. Safe from goroutines.
func (a *App) Dispatch(msg Msg) {
	a.msgs <- msg
}

// SetScreen replaces the active screen and calls its Init.
func (a *App) SetScreen(s Screen) {
	a.current = s
	if cmd := s.Init(); cmd != nil {
		go func() { a.Dispatch(cmd()) }()
	}
}

// Run starts the event loop. It blocks until a QuitMsg is received.
func (a *App) Run() error {
	defer a.screen.Fini()

	// Run initial screen's Init cmd.
	if cmd := a.current.Init(); cmd != nil {
		go func() { a.Dispatch(cmd()) }()
	}

	// Poll tcell events in a background goroutine.
	go func() {
		for {
			ev := a.screen.PollEvent()
			if ev == nil {
				return
			}
			switch ev := ev.(type) {
			case *tcell.EventKey:
				a.Dispatch(KeyMsg{
					Key:  ev.Key(),
					Rune: ev.Rune(),
					Mod:  ev.Modifiers(),
				})
			case *tcell.EventResize:
				w, h := ev.Size()
				a.Dispatch(ResizeMsg{Width: w, Height: h})
			case *tcell.EventInterrupt:
				a.Dispatch(QuitMsg{})
			}
		}
	}()

	// Initial draw.
	w, h := a.screen.Size()
	a.current.Draw(a.screen, w, h)
	a.screen.Show()

	// Main loop.
	for msg := range a.msgs {
		switch msg.(type) {
		case QuitMsg:
			return nil
		}

		// Handle suspend: pause tcell, run function, resume.
		if suspend, ok := msg.(SuspendMsg); ok {
			a.screen.Fini()
			suspend.Fn()
			// Reinitialize screen after suspend
			s, err := tcell.NewScreen()
			if err != nil {
				return fmt.Errorf("tcellapp: reinit screen: %w", err)
			}
			if err := s.Init(); err != nil {
				return fmt.Errorf("tcellapp: reinit screen: %w", err)
			}
			s.EnableMouse()
			a.screen = s
			// Restart event polling
			go func() {
				for {
					ev := a.screen.PollEvent()
					if ev == nil {
						return
					}
					switch ev := ev.(type) {
					case *tcell.EventKey:
						a.Dispatch(KeyMsg{
							Key:  ev.Key(),
							Rune: ev.Rune(),
							Mod:  ev.Modifiers(),
						})
					case *tcell.EventResize:
						w, h := ev.Size()
						a.Dispatch(ResizeMsg{Width: w, Height: h})
					case *tcell.EventInterrupt:
						a.Dispatch(QuitMsg{})
					}
				}
			}()
			w, h := a.screen.Size()
			a.current.Draw(a.screen, w, h)
			a.screen.Show()
			continue
		}

		newScreen, cmd := a.current.HandleMsg(msg)

		// Handle navigation.
		if nav, ok := msg.(NavigateMsg); ok {
			a.SetScreen(nav.Next)
			w, h := a.screen.Size()
			a.current.Draw(a.screen, w, h)
			a.screen.Show()
			continue
		}

		a.current = newScreen

		if cmd != nil {
			go func() { a.Dispatch(cmd()) }()
		}

		w, h := a.screen.Size()
		a.current.Draw(a.screen, w, h)
		a.screen.Show()
	}
	return nil
}

// KeyString returns a canonical string representation of a KeyMsg.
func KeyString(k KeyMsg) string {
	// Handle ctrl modifier with rune keys.
	if k.Key == tcell.KeyRune {
		r := string(k.Rune)
		if k.Mod&tcell.ModCtrl != 0 {
			return "ctrl+" + r
		}
		if k.Mod&tcell.ModAlt != 0 {
			return "alt+" + r
		}
		return r
	}

	name := ""
	switch k.Key {
	case tcell.KeyEnter:
		name = "enter"
	case tcell.KeyEscape:
		name = "esc"
	case tcell.KeyUp:
		name = "up"
	case tcell.KeyDown:
		name = "down"
	case tcell.KeyLeft:
		name = "left"
	case tcell.KeyRight:
		name = "right"
	case tcell.KeyTab:
		name = "tab"
	case tcell.KeyBacktab:
		name = "shift+tab"
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		name = "backspace"
	case tcell.KeyDelete:
		name = "delete"
	case tcell.KeyHome:
		name = "home"
	case tcell.KeyEnd:
		name = "end"
	case tcell.KeyPgUp:
		name = "pgup"
	case tcell.KeyPgDn:
		name = "pgdn"
	case tcell.KeyF1:
		name = "f1"
	case tcell.KeyF2:
		name = "f2"
	case tcell.KeyF3:
		name = "f3"
	case tcell.KeyF4:
		name = "f4"
	case tcell.KeyF5:
		name = "f5"
	case tcell.KeyF6:
		name = "f6"
	case tcell.KeyF7:
		name = "f7"
	case tcell.KeyF8:
		name = "f8"
	case tcell.KeyF9:
		name = "f9"
	case tcell.KeyF10:
		name = "f10"
	case tcell.KeyF11:
		name = "f11"
	case tcell.KeyF12:
		name = "f12"
	case tcell.KeyCtrlA:
		name = "ctrl+a"
	case tcell.KeyCtrlB:
		name = "ctrl+b"
	case tcell.KeyCtrlC:
		name = "ctrl+c"
	case tcell.KeyCtrlD:
		name = "ctrl+d"
	case tcell.KeyCtrlE:
		name = "ctrl+e"
	case tcell.KeyCtrlF:
		name = "ctrl+f"
	case tcell.KeyCtrlG:
		name = "ctrl+g"
	case tcell.KeyCtrlK:
		name = "ctrl+k"
	case tcell.KeyCtrlL:
		name = "ctrl+l"
	case tcell.KeyCtrlN:
		name = "ctrl+n"
	case tcell.KeyCtrlO:
		name = "ctrl+o"
	case tcell.KeyCtrlP:
		name = "ctrl+p"
	case tcell.KeyCtrlQ:
		name = "ctrl+q"
	case tcell.KeyCtrlR:
		name = "ctrl+r"
	case tcell.KeyCtrlS:
		name = "ctrl+s"
	case tcell.KeyCtrlT:
		name = "ctrl+t"
	case tcell.KeyCtrlU:
		name = "ctrl+u"
	case tcell.KeyCtrlV:
		name = "ctrl+v"
	case tcell.KeyCtrlW:
		name = "ctrl+w"
	case tcell.KeyCtrlX:
		name = "ctrl+x"
	case tcell.KeyCtrlY:
		name = "ctrl+y"
	case tcell.KeyCtrlZ:
		name = "ctrl+z"
	default:
		name = fmt.Sprintf("key(%d)", k.Key)
	}

	// Add shift modifier if present and not already in the name.
	if k.Mod&tcell.ModShift != 0 && k.Key != tcell.KeyBacktab {
		name = "shift+" + name
	}

	return name
}
