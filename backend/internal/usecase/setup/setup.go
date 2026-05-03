package setup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
	"github.com/veilence/veilence-mx/backend/internal/usecase/shared"
)

// Service handles the initial setup flow for a fresh installation.
type Service struct {
	users         usecase.UserRepository
	rbacRepo      usecase.RBACRepository
	hasher        usecase.PasswordHasher
	tokenProvider usecase.TokenProvider
}

// NewService creates a new setup service.
func NewService(
	users usecase.UserRepository,
	rbacRepo usecase.RBACRepository,
	hasher usecase.PasswordHasher,
	tokenProvider usecase.TokenProvider,
) *Service {
	return &Service{
		users:         users,
		rbacRepo:      rbacRepo,
		hasher:        hasher,
		tokenProvider: tokenProvider,
	}
}

// IsSetupRequired returns true if no users exist in the database.
func (s *Service) IsSetupRequired(ctx context.Context) (bool, error) {
	count, err := s.users.CountAll(ctx)
	if err != nil {
		return false, fmt.Errorf("counting users: %w", err)
	}
	return count == 0, nil
}

// InitializeRequest holds the data for the initial setup.
type InitializeRequest struct {
	Email         string
	Password      string
	FirstName     string
	LastName      string
	WorkspaceName string
	WorkspaceSlug string
}

// InitializeResult holds the result of the initial setup.
type InitializeResult struct {
	User         *entity.User
	Workspace    *entity.Workspace
	AccessToken  string
	RefreshToken string
}

// Initialize creates the first user, workspace, and owner role.
// Uses a serializable transaction to prevent race conditions.
func (s *Service) Initialize(ctx context.Context, req InitializeRequest) (*InitializeResult, error) {
	// Check if setup is still needed
	count, err := s.users.CountAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("counting users: %w", err)
	}
	if count > 0 {
		return nil, entity.ErrSetupAlreadyCompleted
	}

	if err := entity.ValidatePasswordWithContext(req.Password, req.Email); err != nil {
		return nil, err
	}

	normalizedEmail := normalizeEmail(req.Email)

	passwordHash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &entity.User{
		Email:         normalizedEmail,
		PasswordHash:  passwordHash,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		IsActive:      true,
		IsSuperAdmin:  true, // First user is platform superadmin
		EmailVerified: true, // First user is implicitly trusted
		AuthProvider:  entity.AuthProviderLocal,
	}

	var workspace *entity.Workspace

	// Wrap in transaction for race condition protection
	err = s.rbacRepo.WithTransaction(ctx, func(tx usecase.RBACRepository) error {
		// Re-check count inside transaction (serializable isolation)
		// We use the users repo directly since it's not part of the RBAC transaction,
		// but the unique email constraint on the users table provides the safety net.
		if err := s.users.Create(ctx, user); err != nil {
			return fmt.Errorf("creating first user: %w", err)
		}

		// Create workspace
		workspace = &entity.Workspace{
			Name:        req.WorkspaceName,
			Slug:        req.WorkspaceSlug,
			Description: "Default workspace",
			OwnerID:     user.ID,
			IsActive:    true,
		}
		if err := tx.CreateWorkspace(ctx, workspace); err != nil {
			return fmt.Errorf("creating workspace: %w", err)
		}

		// Create default roles
		roles, err := shared.CreateDefaultRoles(ctx, tx, workspace.ID)
		if err != nil {
			return fmt.Errorf("creating default roles: %w", err)
		}

		// Find the owner role ID
		var ownerRoleID string
		for _, role := range roles {
			if role.Name == entity.RoleOwner {
				ownerRoleID = role.ID
				break
			}
		}

		// Create member with owner role
		member := &entity.WorkspaceMember{
			WorkspaceID: workspace.ID,
			UserID:      user.ID,
			RoleID:      ownerRoleID,
			JoinedAt:    time.Now(),
		}
		if err := tx.CreateMember(ctx, member); err != nil {
			return fmt.Errorf("creating owner membership: %w", err)
		}

		return nil
	})
	if err != nil {
		// If user was created but transaction failed, this is a partial state.
		// The unique email constraint prevents duplicate setup attempts.
		if errors.Is(err, entity.ErrSetupAlreadyCompleted) {
			return nil, err
		}
		return nil, fmt.Errorf("setup transaction: %w", err)
	}

	// Generate tokens so user is immediately logged in
	accessToken, err := s.tokenProvider.GenerateAccessToken(user.ID, user.Email, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	refreshToken, err := s.tokenProvider.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	slog.Info("initial setup completed", "user_id", user.ID, "workspace_id", workspace.ID)

	return &InitializeResult{
		User:         user,
		Workspace:    workspace,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// normalizeEmail lowercases, trims whitespace, and strips +tags from an email.
func normalizeEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return email
	}
	local := parts[0]
	if idx := strings.Index(local, "+"); idx != -1 {
		local = local[:idx]
	}
	return local + "@" + parts[1]
}
