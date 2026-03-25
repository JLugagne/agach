package domain

import "time"

const (
	DefaultRefreshTokenTTL    = 7 * 24 * time.Hour
	DefaultRememberMeTokenTTL = 30 * 24 * time.Hour
	DefaultDaemonJWTTTL       = 30 * 24 * time.Hour
)
