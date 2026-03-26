---
name: doc-identity
description: "Agach identity system: authentication (JWT, bcrypt), SSO/OIDC, teams, users, nodes, daemon onboarding, token management"
user-invocable: true
disable-model-invocation: false
---

# Agach Identity System (`internal/identity/`)

## Overview
Authentication, authorization, SSO, team management, and daemon node onboarding.

## Domain Types

### Core (`domain/types.go`)
- `UserID`, `TeamID` — UUID-based identifiers
- `MemberRole` — "admin" or "member"
- `Team` — ID, Name, Slug, Description, timestamps
- `User` — ID, Email, DisplayName, PasswordHash, SSOProvider, SSOSubject, Role, TeamID, timestamps
- `Actor` — UserID, Email, Role + `IsAdmin()` method (request-scoped auth context)
- `DaemonActor` — NodeID, OwnerUserID, Mode + `IsZero()` method

### Nodes (`domain/node.go`)
- `NodeID`, `OnboardingCodeID` — UUID-based identifiers
- `NodeMode` — "default" or "shared"
- `NodeStatus` — "active" or "revoked"
- `Node` — ID, OwnerUserID, Name, Mode, Status, RefreshTokenHash, LastSeenAt, RevokedAt, timestamps
- `OnboardingCode` — ID, Code (6-digit), CreatedByUserID, NodeMode, NodeName, ExpiresAt, UsedAt, UsedByNodeID
- `NodeAccess` — ID, NodeID, UserID (nullable), TeamID (nullable)

### SSO Config (`domain/ssoconfig.go`)
- `SsoProvider` — Name, Icon, SAML (nullable), OIDC (nullable)
- `OIDCConfig` — IssuerURL, ClientID, ClientSecret, RedirectURL, Scopes
- `SAMLConfig` — MetadataURL, EntityID, ACSURL, Certificate (not yet supported)

### Token TTLs (`domain/ttl.go`)
- DefaultRefreshTokenTTL = 7 days
- DefaultRememberMeTokenTTL = 30 days
- DefaultDaemonJWTTTL = 30 days

### Errors (`domain/errors.go`)
ErrUnauthorized, ErrForbidden, ErrUserNotFound, ErrInvalidCredentials, ErrEmailAlreadyExists,
ErrSSOUserNoPassword, ErrTeamNotFound, ErrTeamSlugConflict, ErrSSOProviderNotFound, ErrSSONotSupported,
ErrNodeNotFound, ErrNodeRevoked, ErrOnboardingCodeNotFound/Expired/Used

## Repository Interfaces
- `UserRepository` — Create, FindByID, FindByEmail, FindBySSO, Update, ListAll, ListByTeam
- `TeamRepository` — Create, FindByID, FindBySlug, List, Update, Delete
- `NodeRepository` — Create, FindByID, ListByOwner, ListActiveByOwner, Update, UpdateLastSeen
- `OnboardingCodeRepository` — Create, FindByCode, MarkUsed (FOR UPDATE lock), DeleteExpired
- `NodeAccessRepository` — GrantUser/Team, RevokeUser/Team, ListByNode, HasAccess (ON CONFLICT DO NOTHING)

## Service Interfaces
- `AuthCommands` — Register, Login, LoginSSO, RefreshToken, Logout, UpdateProfile, ChangePassword, RefreshDaemonToken
- `AuthQueries` — ValidateJWT, ValidateDaemonJWT, GetCurrentUser
- `TeamCommands` — CreateTeam, UpdateTeam, DeleteTeam, AddUserToTeam, RemoveUserFromTeam, SetUserRole
- `TeamQueries` — ListTeams, GetTeam, ListUsers, ListTeamMembers
- `OnboardingCommands` — GenerateCode, CompleteOnboarding
- `NodeCommands` — RevokeNode, UpdateNodeAccess, RenameNode
- `NodeQueries` — ListNodes, GetNode

## App Layer

### Auth (`app/auth.go`)
- bcrypt cost 12, min password 8 chars, min secret 32 bytes
- Access token TTL: 15 minutes
- Refresh token: 7 days (30 days with remember_me)
- JWT claims: sub, email, role, token_type, iat, exp
- Daemon JWT claims: sub (nodeID), owner_id, mode, token_type

### SSO (`app/sso.go`)
- OIDC only (SAML not yet supported)
- Full flow: discovery → code exchange → JWK fetch → ID token validation
- RSA + ECDSA (P-256/P-384/P-521) key support
- Auto-creates user on first SSO login

### Teams (`app/teams.go`)
- All mutations require `actor.IsAdmin()`

### Onboarding (`app/onboarding.go`)
- 6-digit numeric codes with uniqueness retries (3 attempts)
- 15-minute code expiry
- Refresh token: 32 random bytes, bcrypt hashed (cost 12)

### Nodes (`app/nodes.go`)
- All mutations require node ownership (node.OwnerUserID == actor.UserID)
- UpdateNodeAccess requires NodeModeShared

## HTTP Routes

### Auth (rate limited: 5 requests / 15 minutes per IP)
```
POST   /api/auth/register     — Register new user
POST   /api/auth/login        — Login (email/password, remember_me)
POST   /api/auth/refresh      — Refresh access token (refresh_token cookie)
POST   /api/auth/logout       — Clear refresh_token cookie
GET    /api/auth/me           — Get current user
PATCH  /api/auth/me           — Update display name
POST   /api/auth/me/password  — Change password
```

### SSO
```
GET    /api/auth/sso/providers              — List configured providers
GET    /api/auth/sso/{provider}/authorize   — OIDC authorization initiation
GET    /api/auth/sso/{provider}/callback    — OIDC callback (returns #sso_token=...)
```

### Teams
```
GET    /api/identity/teams           — List teams
POST   /api/identity/teams           — Create team (admin)
DELETE /api/identity/teams/{id}      — Delete team (admin)
GET    /api/identity/users           — List users (admins see emails)
PUT    /api/identity/users/{id}/team — Add user to team (admin)
DELETE /api/identity/users/{id}/team — Remove from team (admin)
PUT    /api/identity/users/{id}/role — Set user role (admin)
```

### Onboarding & Nodes
```
POST   /api/onboarding/codes       — Generate onboarding code (auth required)
POST   /api/onboarding/complete    — Complete onboarding (unauthenticated)
POST   /api/daemon/refresh         — Refresh daemon token (unauthenticated)
GET    /api/nodes                  — List nodes (auth required)
GET    /api/nodes/{id}             — Get node (ownership required)
DELETE /api/nodes/{id}             — Revoke node (ownership required)
PATCH  /api/nodes/{id}/name        — Rename node
PUT    /api/nodes/{id}/access      — Update access grants (shared mode only)
```

## Security
- Bcrypt password hashing (cost 12)
- HMAC-SHA256 JWT signing (≥ 32 byte secret)
- pgp_sym_encrypt for sensitive DB columns (password_hash, sso_subject)
- HttpOnly, Secure, SameSite=Strict cookies
- OIDC state verification with HMAC signatures
- Default admin seeded on first run: admin@agach.local / admin (env overridable)

## Init Wiring (`init.go`)
1. Create repositories (runs migrations)
2. Create SSOService if providers configured
3. Wire auth service (with/without nodes)
4. Wire team, onboarding, node services
5. Seed default admin if no users exist
6. Return System struct with all services
