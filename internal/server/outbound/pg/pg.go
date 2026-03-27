package pg

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	agentsrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/agents"
	chatsrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/chats"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/columns"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/comments"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/dependencies"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/dockerfiles"
	featuresrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/features"
	notificationsrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/notifications"
	projectaccessrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projectaccess"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projects"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/skills"
	specializedrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/specialized"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/toolusage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations
var migrationsFS embed.FS

// Repositories holds all PostgreSQL repository implementations.
type Repositories struct {
	Projects          projects.ProjectRepository
	Agents            agentsrepo.AgentRepository
	Tasks             tasks.TaskRepository
	Columns           columns.ColumnRepository
	Comments          comments.CommentRepository
	Dependencies      dependencies.DependencyRepository
	ToolUsage         toolusage.ToolUsageRepository
	Skills            skills.SkillRepository
	Dockerfiles       dockerfiles.DockerfileRepository
	Features          featuresrepo.FeatureRepository
	Notifications     notificationsrepo.NotificationRepository
	Chats             chatsrepo.ChatSessionRepository
	SpecializedAgents specializedrepo.SpecializedAgentRepository
	ProjectAccess     projectaccessrepo.ProjectAccessRepository
}

// NewRepositories creates all repository implementations backed by a pgxpool.Pool and runs migrations.
func NewRepositories(pool *pgxpool.Pool) (*Repositories, error) {
	if pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if _, err := pool.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				filename TEXT PRIMARY KEY,
				applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
			)`); err != nil {
			return nil, fmt.Errorf("creating schema_migrations table: %w", err)
		}

		entries, err := migrationsFS.ReadDir("migrations")
		if err != nil {
			return nil, fmt.Errorf("reading migrations directory: %w", err)
		}
		for _, entry := range entries {
			var alreadyApplied bool
			if err := pool.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)`,
				entry.Name(),
			).Scan(&alreadyApplied); err != nil {
				return nil, fmt.Errorf("checking schema_migrations for %s: %w", entry.Name(), err)
			}
			if alreadyApplied {
				continue
			}

			sql, err := migrationsFS.ReadFile("migrations/" + entry.Name())
			if err != nil {
				return nil, fmt.Errorf("reading migration %s: %w", entry.Name(), err)
			}
			if _, err := pool.Exec(ctx, string(sql)); err != nil {
				return nil, fmt.Errorf("applying migration %s: %w", entry.Name(), err)
			}
			if _, err := pool.Exec(ctx,
				`INSERT INTO schema_migrations (filename) VALUES ($1)`,
				entry.Name(),
			); err != nil {
				return nil, fmt.Errorf("recording schema_migrations for %s: %w", entry.Name(), err)
			}
		}
	}
	base := &baseRepository{pool: pool}
	return &Repositories{
		Projects:          &projectRepository{base},
		Agents:            &agentRepository{base},
		Tasks:             &taskRepository{base},
		Columns:           &columnRepository{base},
		Comments:          &commentRepository{base},
		Dependencies:      &dependencyRepository{base},
		ToolUsage:         &toolUsageRepository{base},
		Skills:            &skillRepository{base},
		Dockerfiles:       &dockerfileRepository{base},
		Features:          &featureRepository{base},
		Notifications:     &notificationRepository{base},
		Chats:             &chatSessionRepository{base},
		SpecializedAgents: &specializedAgentRepository{base},
		ProjectAccess:     &projectAccessRepository{base},
	}, nil
}

const queryTimeout = 30 * time.Second

type baseRepository struct {
	pool *pgxpool.Pool
}

func (r *baseRepository) ctx(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, queryTimeout)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23514"
	}
	return false
}

func jsonMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("[]")
	}
	return b
}

func jsonUnmarshalStrings(data []byte) []string {
	if len(data) == 0 {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal(data, &result); err != nil {
		return []string{}
	}
	if result == nil {
		return []string{}
	}
	return result
}

func newID() string {
	id, _ := uuid.NewV7()
	return id.String()
}

type scanner interface {
	Scan(dest ...any) error
}

// compile-time interface checks
var (
	_ projects.ProjectRepository                 = (*projectRepository)(nil)
	_ agentsrepo.AgentRepository                 = (*agentRepository)(nil)
	_ tasks.TaskRepository                       = (*taskRepository)(nil)
	_ columns.ColumnRepository                   = (*columnRepository)(nil)
	_ comments.CommentRepository                 = (*commentRepository)(nil)
	_ dependencies.DependencyRepository          = (*dependencyRepository)(nil)
	_ toolusage.ToolUsageRepository              = (*toolUsageRepository)(nil)
	_ skills.SkillRepository                     = (*skillRepository)(nil)
	_ featuresrepo.FeatureRepository             = (*featureRepository)(nil)
	_ notificationsrepo.NotificationRepository   = (*notificationRepository)(nil)
	_ chatsrepo.ChatSessionRepository            = (*chatSessionRepository)(nil)
	_ specializedrepo.SpecializedAgentRepository = (*specializedAgentRepository)(nil)
)
