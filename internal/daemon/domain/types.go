package domain

// DaemonState represents the current state of the daemon.
type DaemonState string

const (
	StateDisconnected DaemonState = "disconnected"
	StateOnboarding   DaemonState = "onboarding"
	StateConnected    DaemonState = "connected"
)
