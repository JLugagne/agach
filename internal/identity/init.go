package identity

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/app"
	identitydomain "github.com/JLugagne/agach-mcp/internal/identity/domain"
	identitycmds "github.com/JLugagne/agach-mcp/internal/identity/inbound/commands"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/service"
	"github.com/JLugagne/agach-mcp/internal/identity/outbound/pg"
	identitysvrconfig "github.com/JLugagne/agach-mcp/internal/identity/svrconfig"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// Config holds the configuration for the identity system.
type Config struct {
	Logger    *logrus.Logger
	JWTSecret []byte
	SSO       identitysvrconfig.SsoConfig
}

// System holds the initialized identity services.
type System struct {
	AuthCommands service.AuthCommands
	AuthQueries  service.AuthQueries
	TeamCommands service.TeamCommands
	TeamQueries  service.TeamQueries
	SSOConfig    identitysvrconfig.SsoConfig
	JWTSecret    []byte
}

// Init initializes the identity system: runs migrations, wires repositories and services.
func Init(ctx context.Context, cfg Config, pool *pgxpool.Pool) (*System, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}

	logger.Info("Initializing identity system")

	repos, err := pg.NewRepositories(ctx, pool, string(cfg.JWTSecret))
	if err != nil {
		logger.WithError(err).Error("Failed to initialize identity repositories")
		return nil, err
	}

	logger.Info("Identity repositories initialized")

	var ssoSvc *app.SSOService
	if len(cfg.SSO.Providers) > 0 {
		ssoSvc = app.NewSSOService(cfg.SSO, repos.Users, cfg.JWTSecret)
	}
	authCmds := app.NewAuthService(repos.Users, repos.APIKeys, cfg.JWTSecret, ssoSvc)
	authQrys := app.NewAuthQueriesService(repos.Users, repos.APIKeys, cfg.JWTSecret, ssoSvc)
	teamCmds := app.NewTeamService(repos.Teams, repos.Users)
	teamQrys := app.NewTeamQueriesService(repos.Teams, repos.Users)

	if err := seedDefaultAdmin(ctx, repos, logger); err != nil {
		logger.WithError(err).Error("Failed to seed default admin user")
		return nil, err
	}

	return &System{
		AuthCommands: authCmds,
		AuthQueries:  authQrys,
		TeamCommands: teamCmds,
		TeamQueries:  teamQrys,
		SSOConfig:    cfg.SSO,
		JWTSecret:    cfg.JWTSecret,
	}, nil
}

// seedDefaultAdmin creates an admin user if no users exist yet.
// Credentials are read from AGACH_ADMIN_USER / AGACH_ADMIN_PASSWORD env vars,
// defaulting to admin / admin.
func seedDefaultAdmin(ctx context.Context, repos *pg.Repositories, logger *logrus.Logger) error {
	existing, err := repos.Users.ListAll(ctx)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil // already seeded
	}

	email := os.Getenv("AGACH_ADMIN_USER")
	if email == "" {
		email = "admin@agach.local"
	} else if !strings.Contains(email, "@") {
		email = email + "@agach.local"
	}
	password := os.Getenv("AGACH_ADMIN_PASSWORD")
	if password == "" {
		password = "admin"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return err
	}

	now := time.Now()
	user := identitydomain.User{
		ID:           identitydomain.NewUserID(),
		Email:        email,
		DisplayName:  "Admin",
		PasswordHash: string(hash),
		Role:         identitydomain.RoleAdmin,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := repos.Users.Create(ctx, user); err != nil {
		return err
	}

	logger.WithField("email", email).Warn("Default admin user created — change this password!")
	return nil
}

// RegisterRoutes registers all identity HTTP routes on the given router.
func (s *System) RegisterRoutes(router *mux.Router, ctrl *controller.Controller) {
	authH := identitycmds.NewAuthCommandsHandler(s.AuthCommands, s.AuthQueries, ctrl)
	authH.RegisterRoutes(router)

	teamsH := identitycmds.NewTeamsHandler(s.TeamCommands, s.TeamQueries, s.AuthQueries, ctrl)
	teamsH.RegisterRoutes(router)

	if len(s.SSOConfig.Providers) > 0 {
		ssoH := identitycmds.NewSSOCommandsHandler(s.AuthCommands, s.AuthQueries, ctrl, s.SSOConfig, s.JWTSecret)
		ssoH.RegisterRoutes(router)
	}
}
