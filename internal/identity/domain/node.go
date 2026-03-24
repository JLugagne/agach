package domain

import (
	"time"

	"github.com/google/uuid"
)

type NodeID uuid.UUID

func NewNodeID() NodeID          { id, _ := uuid.NewV7(); return NodeID(id) }
func (id NodeID) String() string { return uuid.UUID(id).String() }

func ParseNodeID(s string) (NodeID, error) {
	id, err := uuid.Parse(s)
	return NodeID(id), err
}

type OnboardingCodeID uuid.UUID

func NewOnboardingCodeID() OnboardingCodeID { id, _ := uuid.NewV7(); return OnboardingCodeID(id) }
func (id OnboardingCodeID) String() string  { return uuid.UUID(id).String() }

func ParseOnboardingCodeID(s string) (OnboardingCodeID, error) {
	id, err := uuid.Parse(s)
	return OnboardingCodeID(id), err
}

// NodeMode represents the access mode of a node.
type NodeMode string

const (
	NodeModeDefault NodeMode = "default"
	NodeModeShared  NodeMode = "shared"
)

// NodeStatus represents the lifecycle status of a node.
type NodeStatus string

const (
	NodeStatusActive  NodeStatus = "active"
	NodeStatusRevoked NodeStatus = "revoked"
)

// Node represents a registered daemon instance.
type Node struct {
	ID               NodeID
	OwnerUserID      UserID
	Name             string
	Mode             NodeMode
	Status           NodeStatus
	RefreshTokenHash string
	LastSeenAt       *time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (n Node) IsActive() bool  { return n.Status == NodeStatusActive }
func (n Node) IsRevoked() bool { return n.Status == NodeStatusRevoked }

// OnboardingCode represents a code used to onboard a daemon.
type OnboardingCode struct {
	ID              OnboardingCodeID
	Code            string
	CreatedByUserID UserID
	NodeMode        NodeMode
	NodeName        string
	ExpiresAt       time.Time
	UsedAt          *time.Time
	UsedByNodeID    *NodeID
	CreatedAt       time.Time
}

func (c OnboardingCode) IsExpired() bool { return time.Now().After(c.ExpiresAt) }
func (c OnboardingCode) IsUsed() bool    { return c.UsedAt != nil }
func (c OnboardingCode) IsValid() bool   { return !c.IsExpired() && !c.IsUsed() }

// NodeAccess represents access granted to a user or team for a shared node.
type NodeAccess struct {
	ID        uuid.UUID
	NodeID    NodeID
	UserID    *UserID
	TeamID    *TeamID
	CreatedAt time.Time
}
