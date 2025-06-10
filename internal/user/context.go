package user

import (
	"context"
	"errors"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

const (
	// UserContextKey is the key for storing user information in context
	UserContextKey contextKey = "user"
)

// ErrNoUserInContext is returned when no user is found in context
var ErrNoUserInContext = errors.New("no user found in context")

// UserInfo represents authenticated user information
type UserInfo struct {
	ID       string   `json:"id"`
	Email    string   `json:"email"`
	Username string   `json:"username"`
	Roles    []string `json:"roles,omitempty"`
}

// FromContext extracts user information from context
func FromContext(ctx context.Context) (*UserInfo, error) {
	user, ok := ctx.Value(UserContextKey).(*UserInfo)
	if !ok || user == nil {
		return nil, ErrNoUserInContext
	}
	return user, nil
}

// MustFromContext extracts user information from context or panics
func MustFromContext(ctx context.Context) *UserInfo {
	user, err := FromContext(ctx)
	if err != nil {
		panic(err)
	}
	return user
}

// WithUser adds user information to context
func WithUser(ctx context.Context, user *UserInfo) context.Context {
	return context.WithValue(ctx, UserContextKey, user)
}

// GetUserID extracts user ID from context, returns empty string if not found
func GetUserID(ctx context.Context) string {
	user, err := FromContext(ctx)
	if err != nil {
		return ""
	}
	return user.ID
}

// RequireUserID extracts user ID from context, returns error if not found
func RequireUserID(ctx context.Context) (string, error) {
	user, err := FromContext(ctx)
	if err != nil {
		return "", err
	}
	if user.ID == "" {
		return "", errors.New("user ID is empty")
	}
	return user.ID, nil
}
