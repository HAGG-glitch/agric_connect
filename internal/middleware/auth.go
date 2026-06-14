package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/agriconnect-ai/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const ContextKeyUser = "auth_user"

type AuthUser struct {
	ID                uuid.UUID
	FullName          string
	PhoneNumber       string
	District          string
	PreferredLanguage string
	Role              string
	IsActive          bool
}

func AuthRequired(accessSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := extractUser(c, accessSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}
		c.Set(ContextKeyUser, user)
		c.Set("user_id", user.ID.String())
		c.Set("user_role", user.Role)
		c.Next()
	}
}

func OptionalAuth(accessSecret string, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := extractUser(c, accessSecret)
		if err == nil && user != nil {
			c.Set(ContextKeyUser, user)
			c.Set("user_id", user.ID.String())
			c.Set("user_role", user.Role)
		}
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRaw, exists := c.Get(ContextKeyUser)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}
		user, ok := userRaw.(*AuthUser)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}

		for _, role := range roles {
			if user.Role == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	}
}

func extractUser(c *gin.Context, accessSecret string) (*AuthUser, error) {
	accessTokenStr, err := c.Cookie("access_token")
	if err != nil || accessTokenStr == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			accessTokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	if accessTokenStr == "" {
		return nil, errors.New("no token")
	}

	claims, err := auth.ValidateToken(accessTokenStr, accessSecret)
	if err != nil {
		return nil, err
	}

	return &AuthUser{
		ID:   claims.UserID,
		Role: claims.Role,
	}, nil
}
