package domain

import (
	"time"

	"github.com/google/uuid"
)

// EncryptedString is a named string type that signals the value should be
// treated as sensitive and stored with at-rest encryption.
type EncryptedString = string

type (
	UserID   uuid.UUID
	TeamID   uuid.UUID
	APIKeyID uuid.UUID
)

func NewUserID() UserID     { id, _ := uuid.NewV7(); return UserID(id) }
func NewTeamID() TeamID     { id, _ := uuid.NewV7(); return TeamID(id) }
func NewAPIKeyID() APIKeyID { id, _ := uuid.NewV7(); return APIKeyID(id) }

func (id UserID) String() string   { return uuid.UUID(id).String() }
func (id TeamID) String() string   { return uuid.UUID(id).String() }
func (id APIKeyID) String() string { return uuid.UUID(id).String() }

func ParseUserID(s string) (UserID, error) {
	id, err := uuid.Parse(s)
	return UserID(id), err
}

func ParseTeamID(s string) (TeamID, error) {
	id, err := uuid.Parse(s)
	return TeamID(id), err
}

func ParseAPIKeyID(s string) (APIKeyID, error) {
	id, err := uuid.Parse(s)
	return APIKeyID(id), err
}

// MemberRole represents a user's role in the system.
type MemberRole string

const (
	RoleAdmin  MemberRole = "admin"
	RoleMember MemberRole = "member"
)

// Team groups users for collaboration.
type Team struct {
	ID          TeamID
	Name        string
	Slug        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// User represents an authenticated user.
type User struct {
	ID           UserID
	Email        string
	DisplayName  string
	PasswordHash string     // bcrypt, empty if SSO-only
	SSOProvider  string     // e.g. "google", "github"
	SSOSubject   string     // provider's sub/user ID
	Role         MemberRole // system-wide role
	TeamID       *TeamID    // optional team membership
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// APIKey represents a programmatic access key (used by the TUI and agents).
type APIKey struct {
	ID         APIKeyID
	UserID     UserID
	Name       string
	KeyHash    string     // SHA-256 of raw key, hex-encoded
	Scopes     []string
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time
	RevokedAt  *time.Time
}

// Actor represents the authenticated caller for a request.
type Actor struct {
	UserID UserID
	Email  string
	Role   MemberRole
}

func (a Actor) IsAdmin() bool { return a.Role == RoleAdmin }
func (a Actor) IsZero() bool  { return a.UserID == UserID{} }
