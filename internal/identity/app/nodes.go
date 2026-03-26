package app

import (
	"context"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/nodeaccess"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/nodes"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/service"
)

type nodeService struct {
	nodes      nodes.NodeRepository
	nodeAccess nodeaccess.NodeAccessRepository
}

func NewNodeService(n nodes.NodeRepository, na nodeaccess.NodeAccessRepository) service.NodeCommands {
	return &nodeService{nodes: n, nodeAccess: na}
}

func NewNodeQueriesService(n nodes.NodeRepository, na nodeaccess.NodeAccessRepository) service.NodeQueries {
	return &nodeService{nodes: n, nodeAccess: na}
}

var (
	_ service.NodeCommands = (*nodeService)(nil)
	_ service.NodeQueries  = (*nodeService)(nil)
)

func (s *nodeService) ListNodes(ctx context.Context, actor domain.Actor) ([]domain.Node, error) {
	return s.nodes.ListByOwner(ctx, actor.UserID)
}

func (s *nodeService) GetNode(ctx context.Context, actor domain.Actor, nodeID domain.NodeID) (domain.Node, error) {
	node, err := s.nodes.FindByID(ctx, nodeID)
	if err != nil {
		return domain.Node{}, err
	}
	if node.OwnerUserID != actor.UserID {
		return domain.Node{}, domain.ErrForbidden
	}
	return node, nil
}

func (s *nodeService) RevokeNode(ctx context.Context, actor domain.Actor, nodeID domain.NodeID) error {
	node, err := s.nodes.FindByID(ctx, nodeID)
	if err != nil {
		return err
	}
	if node.OwnerUserID != actor.UserID {
		return domain.ErrForbidden
	}
	if node.IsRevoked() {
		return domain.ErrNodeRevoked
	}
	now := time.Now()
	node.Status = domain.NodeStatusRevoked
	node.RevokedAt = &now
	node.RefreshTokenHash = ""
	node.UpdatedAt = now
	return s.nodes.Update(ctx, node)
}

func (s *nodeService) UpdateNodeAccess(ctx context.Context, actor domain.Actor, nodeID domain.NodeID, grantUserIDs []domain.UserID, grantTeamIDs []domain.TeamID, revokeUserIDs []domain.UserID, revokeTeamIDs []domain.TeamID) error {
	node, err := s.nodes.FindByID(ctx, nodeID)
	if err != nil {
		return err
	}
	if node.OwnerUserID != actor.UserID {
		return domain.ErrForbidden
	}
	if node.Mode != domain.NodeModeShared {
		return &domain.Error{Code: "NODE_NOT_SHARED", Message: "node access control requires shared mode"}
	}
	for _, uid := range grantUserIDs {
		if err := s.nodeAccess.GrantUser(ctx, nodeID, uid); err != nil {
			return err
		}
	}
	for _, tid := range grantTeamIDs {
		if err := s.nodeAccess.GrantTeam(ctx, nodeID, tid); err != nil {
			return err
		}
	}
	for _, uid := range revokeUserIDs {
		if err := s.nodeAccess.RevokeUser(ctx, nodeID, uid); err != nil {
			return err
		}
	}
	for _, tid := range revokeTeamIDs {
		if err := s.nodeAccess.RevokeTeam(ctx, nodeID, tid); err != nil {
			return err
		}
	}
	return nil
}

func (s *nodeService) ListAllNodes(ctx context.Context, actor domain.Actor) ([]domain.Node, error) {
	if !actor.IsAdmin() {
		return nil, domain.ErrForbidden
	}
	return s.nodes.ListAll(ctx)
}

func (s *nodeService) AdminRevokeNode(ctx context.Context, actor domain.Actor, nodeID domain.NodeID) error {
	if !actor.IsAdmin() {
		return domain.ErrForbidden
	}
	node, err := s.nodes.FindByID(ctx, nodeID)
	if err != nil {
		return err
	}
	if node.IsRevoked() {
		return domain.ErrNodeRevoked
	}
	now := time.Now()
	node.Status = domain.NodeStatusRevoked
	node.RevokedAt = &now
	node.RefreshTokenHash = ""
	node.UpdatedAt = now
	return s.nodes.Update(ctx, node)
}

func (s *nodeService) RenameNode(ctx context.Context, actor domain.Actor, nodeID domain.NodeID, name string) error {
	node, err := s.nodes.FindByID(ctx, nodeID)
	if err != nil {
		return err
	}
	if node.OwnerUserID != actor.UserID {
		return domain.ErrForbidden
	}
	node.Name = name
	node.UpdatedAt = time.Now()
	return s.nodes.Update(ctx, node)
}
