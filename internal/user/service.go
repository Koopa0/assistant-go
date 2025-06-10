package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// Service handles user-related business logic
type Service struct {
	queries *sqlc.Queries
	logger  *slog.Logger
	metrics *observability.Metrics
}

// NewUserService creates a new user service
func NewUserService(queries *sqlc.Queries, logger *slog.Logger, metrics *observability.Metrics) *Service {
	return &Service{
		queries: queries,
		logger:  observability.ServerLogger(logger, "users"),
		metrics: metrics,
	}
}

// UserProfile represents a user's public profile information
type UserProfile struct {
	ID          string                 `json:"id"`
	Username    string                 `json:"username"`
	Email       string                 `json:"email"`
	FullName    *string                `json:"full_name"`
	AvatarURL   *string                `json:"avatar_url"`
	Preferences map[string]interface{} `json:"preferences"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// UserStatistics represents comprehensive user statistics
type UserStatistics struct {
	UserProfile
	TotalConversations int `json:"total_conversations"`
	TotalToolsUsed     int `json:"total_tools_used"`
	TotalTokensUsed    int `json:"total_tokens_used"`
	TotalCostCents     int `json:"total_cost_cents"`
}

// UserActivitySummary represents user activity metrics
type UserActivitySummary struct {
	ID                  string     `json:"id"`
	ConversationsCount  int        `json:"conversations_count"`
	MessagesCount       int        `json:"messages_count"`
	ToolExecutionsCount int        `json:"tool_executions_count"`
	LastActivityAt      *time.Time `json:"last_activity_at"`
	DaysSinceSignup     int        `json:"days_since_signup"`
}

// UserSettings represents user preferences and settings
type UserSettings struct {
	Language                   *string `json:"language"`
	Theme                      *string `json:"theme"`
	DefaultProgrammingLanguage *string `json:"default_programming_language"`
	EmailNotifications         *bool   `json:"email_notifications"`
	Timezone                   *string `json:"timezone"`
}

// CreateUserRequest represents a request to create a new user
type CreateUserRequest struct {
	Username    string                 `json:"username"`
	Email       string                 `json:"email"`
	Password    string                 `json:"password"`
	FullName    *string                `json:"full_name"`
	AvatarURL   *string                `json:"avatar_url"`
	Preferences map[string]interface{} `json:"preferences,omitempty"`
}

// UpdateProfileRequest represents a request to update user profile
type UpdateProfileRequest struct {
	FullName  *string `json:"full_name"`
	AvatarURL *string `json:"avatar_url"`
}

// UpdatePreferencesRequest represents a request to update user preferences
type UpdatePreferencesRequest struct {
	Preferences map[string]interface{} `json:"preferences"`
}

// ChangePasswordRequest represents a request to change user password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// APIKeyRequest represents a request to create an API key
type APIKeyRequest struct {
	Name        string     `json:"name"`
	Permissions []string   `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// APIKeyResponse represents an API key with masked key
type APIKeyResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Key         string     `json:"key"` // Only shown on creation
	MaskedKey   string     `json:"masked_key,omitempty"`
	Permissions []string   `json:"permissions"`
	Status      string     `json:"status"`
	UsageCount  int        `json:"usage_count"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// CreateUser creates a new user account
func (s *Service) CreateUser(ctx context.Context, req *CreateUserRequest) (*UserProfile, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash password", slog.Any("error", err))
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Set default preferences if none provided
	preferences := req.Preferences
	if preferences == nil {
		preferences = map[string]interface{}{
			"language":                   "zh-TW",
			"theme":                      "light",
			"defaultProgrammingLanguage": "Go",
			"emailNotifications":         true,
			"timezone":                   "Asia/Taipei",
		}
	}

	// Marshal preferences to JSON bytes
	prefBytes, err := json.Marshal(preferences)
	if err != nil {
		s.logger.Error("Failed to marshal preferences", slog.Any("error", err))
		return nil, fmt.Errorf("failed to marshal preferences: %w", err)
	}

	// Convert string pointers to pgtype.Text
	var fullNameText pgtype.Text
	if req.FullName != nil {
		fullNameText = pgtype.Text{String: *req.FullName, Valid: true}
	}

	var avatarURLText pgtype.Text
	if req.AvatarURL != nil {
		avatarURLText = pgtype.Text{String: *req.AvatarURL, Valid: true}
	}

	// Create user in database
	user, err := s.queries.CreateUser(ctx, sqlc.CreateUserParams{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FullName:     fullNameText,
		AvatarUrl:    avatarURLText,
		Preferences:  prefBytes,
	})
	if err != nil {
		s.logger.Error("Failed to create user",
			slog.String("email", req.Email),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("User created successfully",
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email))

	return s.createUserRowToProfile(*user), nil
}

// GetUserByID retrieves a user by their ID
func (s *Service) GetUserByID(ctx context.Context, userID string) (*UserProfile, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := s.queries.GetUserByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		s.logger.Error("Failed to get user by ID",
			slog.String("user_id", userID),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return s.getUserByIDRowToProfile(*user), nil
}

// GetUserByEmail retrieves a user by their email
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*UserProfile, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		s.logger.Error("Failed to get user by email",
			slog.String("email", email),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return s.userToProfile(*user), nil
}

// UpdateProfile updates a user's profile information
func (s *Service) UpdateProfile(ctx context.Context, userID string, req *UpdateProfileRequest) (*UserProfile, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Convert string pointers to pgtype.Text
	var fullNameText pgtype.Text
	if req.FullName != nil {
		fullNameText = pgtype.Text{String: *req.FullName, Valid: true}
	}

	var avatarURLText pgtype.Text
	if req.AvatarURL != nil {
		avatarURLText = pgtype.Text{String: *req.AvatarURL, Valid: true}
	}

	user, err := s.queries.UpdateUserProfile(ctx, sqlc.UpdateUserProfileParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		FullName:  fullNameText,
		AvatarUrl: avatarURLText,
	})
	if err != nil {
		s.logger.Error("Failed to update user profile",
			slog.String("user_id", userID),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	s.logger.Info("User profile updated",
		slog.String("user_id", userID))

	return s.updateUserProfileRowToProfile(*user), nil
}

// UpdatePreferences updates a user's preferences
func (s *Service) UpdatePreferences(ctx context.Context, userID string, req *UpdatePreferencesRequest) (*UserProfile, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Marshal preferences to JSON bytes
	prefBytes, err := json.Marshal(req.Preferences)
	if err != nil {
		s.logger.Error("Failed to marshal preferences", slog.Any("error", err))
		return nil, fmt.Errorf("failed to marshal preferences: %w", err)
	}

	user, err := s.queries.UpdateUserPreferences(ctx, sqlc.UpdateUserPreferencesParams{
		ID:          pgtype.UUID{Bytes: id, Valid: true},
		Preferences: prefBytes,
	})
	if err != nil {
		s.logger.Error("Failed to update user preferences",
			slog.String("user_id", userID),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to update preferences: %w", err)
	}

	s.logger.Info("User preferences updated",
		slog.String("user_id", userID))

	return s.updateUserPreferencesRowToProfile(*user), nil
}

// GetUserStatistics retrieves comprehensive user statistics
func (s *Service) GetUserStatistics(ctx context.Context, userID string) (*UserStatistics, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	stats, err := s.queries.GetUserStatistics(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		s.logger.Error("Failed to get user statistics",
			slog.String("user_id", userID),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	// Convert pgtype.Text to *string
	var fullName *string
	if stats.FullName.Valid {
		fullName = &stats.FullName.String
	}

	return &UserStatistics{
		UserProfile: UserProfile{
			ID:        uuid.UUID(stats.ID.Bytes).String(),
			Username:  stats.Username,
			Email:     stats.Email,
			FullName:  fullName,
			CreatedAt: stats.CreatedAt,
		},
		TotalConversations: int(stats.TotalConversations),
		TotalToolsUsed:     int(stats.TotalToolsUsed),
		TotalTokensUsed:    int(stats.TotalTokensUsed),
		TotalCostCents:     int(stats.TotalCostCents),
	}, nil
}

// GetUserActivitySummary retrieves user activity summary
func (s *Service) GetUserActivitySummary(ctx context.Context, userID string) (*UserActivitySummary, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	summary, err := s.queries.GetUserActivitySummary(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		s.logger.Error("Failed to get user activity summary",
			slog.String("user_id", userID),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to get activity summary: %w", err)
	}

	var lastActivityAt *time.Time
	if lastActivity, ok := summary.LastActivityAt.(time.Time); ok {
		lastActivityAt = &lastActivity
	}

	return &UserActivitySummary{
		ID:                  summary.ID.String(),
		ConversationsCount:  int(summary.ConversationsCount),
		MessagesCount:       int(summary.MessagesCount),
		ToolExecutionsCount: int(summary.ToolExecutionsCount),
		LastActivityAt:      lastActivityAt,
		DaysSinceSignup:     int(summary.DaysSinceSignup),
	}, nil
}

// GetUserSettings retrieves user settings and preferences
func (s *Service) GetUserSettings(ctx context.Context, userID string) (*UserSettings, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	settings, err := s.queries.GetUserSettings(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		s.logger.Error("Failed to get user settings",
			slog.String("user_id", userID),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// Convert strings to *string for optional fields
	var language, theme, defaultLang, timezone *string
	if settings.Language != "" {
		language = &settings.Language
	}
	if settings.Theme != "" {
		theme = &settings.Theme
	}
	if settings.DefaultProgrammingLanguage != "" {
		defaultLang = &settings.DefaultProgrammingLanguage
	}
	if settings.Timezone != "" {
		timezone = &settings.Timezone
	}

	var emailNotifs *bool
	emailNotifs = &settings.EmailNotifications

	return &UserSettings{
		Language:                   language,
		Theme:                      theme,
		DefaultProgrammingLanguage: defaultLang,
		EmailNotifications:         emailNotifs,
		Timezone:                   timezone,
	}, nil
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(ctx context.Context, userID string, req *ChangePasswordRequest) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Get current user basic info first
	userInfo, err := s.queries.GetUserByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Get full user record with password hash for verification
	user, err := s.queries.GetUserByEmail(ctx, userInfo.Email)
	if err != nil {
		return fmt.Errorf("failed to get user with password: %w", err)
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		s.logger.Warn("Invalid current password attempt",
			slog.String("user_id", userID))
		return fmt.Errorf("invalid current password")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash new password", slog.Any("error", err))
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	_, err = s.queries.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:           pgtype.UUID{Bytes: id, Valid: true},
		PasswordHash: string(hashedPassword),
	})
	if err != nil {
		s.logger.Error("Failed to update password",
			slog.String("user_id", userID),
			slog.Any("error", err))
		return fmt.Errorf("failed to update password: %w", err)
	}

	s.logger.Info("Password changed successfully",
		slog.String("user_id", userID))

	return nil
}

// AddFavoriteTool adds a tool to user's favorites
func (s *Service) AddFavoriteTool(ctx context.Context, userID, toolID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	_, err = s.queries.AddFavoriteTool(ctx, sqlc.AddFavoriteToolParams{
		ID:      pgtype.UUID{Bytes: id, Valid: true},
		Column2: []byte(fmt.Sprintf(`"%s"`, toolID)),
	})
	if err != nil {
		s.logger.Error("Failed to add favorite tool",
			slog.String("user_id", userID),
			slog.String("tool_id", toolID),
			slog.Any("error", err))
		return fmt.Errorf("failed to add favorite tool: %w", err)
	}

	s.logger.Info("Favorite tool added",
		slog.String("user_id", userID),
		slog.String("tool_id", toolID))

	return nil
}

// RemoveFavoriteTool removes a tool from user's favorites
func (s *Service) RemoveFavoriteTool(ctx context.Context, userID, toolID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	_, err = s.queries.RemoveFavoriteTool(ctx, sqlc.RemoveFavoriteToolParams{
		ID:      pgtype.UUID{Bytes: id, Valid: true},
		Column2: toolID,
	})
	if err != nil {
		s.logger.Error("Failed to remove favorite tool",
			slog.String("user_id", userID),
			slog.String("tool_id", toolID),
			slog.Any("error", err))
		return fmt.Errorf("failed to remove favorite tool: %w", err)
	}

	s.logger.Info("Favorite tool removed",
		slog.String("user_id", userID),
		slog.String("tool_id", toolID))

	return nil
}

// DeactivateUser deactivates a user account
func (s *Service) DeactivateUser(ctx context.Context, userID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	_, err = s.queries.DeactivateUser(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		s.logger.Error("Failed to deactivate user",
			slog.String("user_id", userID),
			slog.Any("error", err))
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	s.logger.Info("User deactivated",
		slog.String("user_id", userID))

	return nil
}

// generateAPIKey generates a random API key
func (s *Service) generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// userToProfile converts a database user to a UserProfile
func (s *Service) userToProfile(user sqlc.User) *UserProfile {
	// Parse preferences from JSONB bytes
	var preferences map[string]interface{}
	if user.Preferences != nil {
		if err := json.Unmarshal(user.Preferences, &preferences); err != nil {
			s.logger.Warn("Failed to parse user preferences", slog.Any("error", err))
			preferences = make(map[string]interface{})
		}
	} else {
		preferences = make(map[string]interface{})
	}

	// Convert pgtype.Text to *string
	var fullName *string
	if user.FullName.Valid {
		fullName = &user.FullName.String
	}

	var avatarURL *string
	if user.AvatarUrl.Valid {
		avatarURL = &user.AvatarUrl.String
	}

	return &UserProfile{
		ID:          uuid.UUID(user.ID.Bytes).String(),
		Username:    user.Username,
		Email:       user.Email,
		FullName:    fullName,
		AvatarURL:   avatarURL,
		Preferences: preferences,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

// createUserRowToProfile converts a CreateUserRow to a UserProfile
func (s *Service) createUserRowToProfile(user sqlc.CreateUserRow) *UserProfile {
	// Parse preferences from JSONB bytes
	var preferences map[string]interface{}
	if user.Preferences != nil {
		if err := json.Unmarshal(user.Preferences, &preferences); err != nil {
			s.logger.Warn("Failed to parse user preferences", slog.Any("error", err))
			preferences = make(map[string]interface{})
		}
	} else {
		preferences = make(map[string]interface{})
	}

	// Convert pgtype.Text to *string
	var fullName *string
	if user.FullName.Valid {
		fullName = &user.FullName.String
	}

	var avatarURL *string
	if user.AvatarUrl.Valid {
		avatarURL = &user.AvatarUrl.String
	}

	return &UserProfile{
		ID:          uuid.UUID(user.ID.Bytes).String(),
		Username:    user.Username,
		Email:       user.Email,
		FullName:    fullName,
		AvatarURL:   avatarURL,
		Preferences: preferences,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

// getUserByIDRowToProfile converts a GetUserByIDRow to a UserProfile
func (s *Service) getUserByIDRowToProfile(user sqlc.GetUserByIDRow) *UserProfile {
	// Parse preferences from JSONB bytes
	var preferences map[string]interface{}
	if user.Preferences != nil {
		if err := json.Unmarshal(user.Preferences, &preferences); err != nil {
			s.logger.Warn("Failed to parse user preferences", slog.Any("error", err))
			preferences = make(map[string]interface{})
		}
	} else {
		preferences = make(map[string]interface{})
	}

	// Convert pgtype.Text to *string
	var fullName *string
	if user.FullName.Valid {
		fullName = &user.FullName.String
	}

	var avatarURL *string
	if user.AvatarUrl.Valid {
		avatarURL = &user.AvatarUrl.String
	}

	return &UserProfile{
		ID:          uuid.UUID(user.ID.Bytes).String(),
		Username:    user.Username,
		Email:       user.Email,
		FullName:    fullName,
		AvatarURL:   avatarURL,
		Preferences: preferences,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

// updateUserProfileRowToProfile converts an UpdateUserProfileRow to a UserProfile
func (s *Service) updateUserProfileRowToProfile(user sqlc.UpdateUserProfileRow) *UserProfile {
	// Parse preferences from JSONB bytes
	var preferences map[string]interface{}
	if user.Preferences != nil {
		if err := json.Unmarshal(user.Preferences, &preferences); err != nil {
			s.logger.Warn("Failed to parse user preferences", slog.Any("error", err))
			preferences = make(map[string]interface{})
		}
	} else {
		preferences = make(map[string]interface{})
	}

	// Convert pgtype.Text to *string
	var fullName *string
	if user.FullName.Valid {
		fullName = &user.FullName.String
	}

	var avatarURL *string
	if user.AvatarUrl.Valid {
		avatarURL = &user.AvatarUrl.String
	}

	return &UserProfile{
		ID:          uuid.UUID(user.ID.Bytes).String(),
		Username:    user.Username,
		Email:       user.Email,
		FullName:    fullName,
		AvatarURL:   avatarURL,
		Preferences: preferences,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

// updateUserPreferencesRowToProfile converts an UpdateUserPreferencesRow to a UserProfile
func (s *Service) updateUserPreferencesRowToProfile(user sqlc.UpdateUserPreferencesRow) *UserProfile {
	// Parse preferences from JSONB bytes
	var preferences map[string]interface{}
	if user.Preferences != nil {
		if err := json.Unmarshal(user.Preferences, &preferences); err != nil {
			s.logger.Warn("Failed to parse user preferences", slog.Any("error", err))
			preferences = make(map[string]interface{})
		}
	} else {
		preferences = make(map[string]interface{})
	}

	// Convert pgtype.Text to *string
	var fullName *string
	if user.FullName.Valid {
		fullName = &user.FullName.String
	}

	var avatarURL *string
	if user.AvatarUrl.Valid {
		avatarURL = &user.AvatarUrl.String
	}

	return &UserProfile{
		ID:          uuid.UUID(user.ID.Bytes).String(),
		Username:    user.Username,
		Email:       user.Email,
		FullName:    fullName,
		AvatarURL:   avatarURL,
		Preferences: preferences,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}
